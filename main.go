package main

import (
	"fmt"
	"net/http"
	"log"
	"strings"
	"math"
	"regexp"
	"io/ioutil"
	"strconv"
	"encoding/json"
	"database/sql"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

// 東京駅の座標
const (
	TokyoStationX = 139.7673068 // 東京駅の経度
	TokyoStationY = 35.6809591  // 東京駅の緯度
	EarthRadius   = 6371.0      // 地球の半径(km)
)

// 郵便番号から該当する住所を取得するAPIのレスポンス構造体
type AddressResponse struct {
	PostalCode       string  `json:"postal_code"`        // リクエストパラメータで与えた郵便番号
	HitCount         int     `json:"hit_count"`          // 該当する地域の数
	Address          string  `json:"address"`            // 外部APIから取得した各住所のうち、共通する部分の住所
	TokyoStaDistance float64 `json:"tokyo_sta_distance"` // 外部APIから取得した各住所のうち、東京駅から最も離れている地域から東京駅までの距離(km)
}

// 外部APIのレスポンス構造体
type ExternalAPIResponse struct {
	Response struct {
		Location []struct {
			City       string `json:"city"`
			CityKana   string `json:"city_kana"`
			Town       string `json:"town"`
			TownKana   string `json:"town_kana"`
			X          string `json:"x"`
			Y          string `json:"y"`
			Prefecture string `json:"prefecture"`
			Postal     string `json:"postal"`
		} `json:"location"`
	} `json:"response"`
}

// アクセスログを取得するAPI用の構造体
type AccessLogItem struct {
	PostalCode   string `json:"postal_code"`
	RequestCount int    `json:"request_count"`
}

type AccessLogsResponse struct {
	AccessLogs []AccessLogItem `json:"access_logs"`
}

// 各住所のうち、共通部分の住所を取得
func getCommonAddress(locations []struct {
	City       string `json:"city"`
	CityKana   string `json:"city_kana"`
	Town       string `json:"town"`
	TownKana   string `json:"town_kana"`
	X          string `json:"x"`
	Y          string `json:"y"`
	Prefecture string `json:"prefecture"`
	Postal     string `json:"postal"`
}) string {
	if len(locations) == 0 {
		return ""
	}
	
	// 各住所のうち、最初の住所を基準にする
	prefecture := locations[0].Prefecture
	city := locations[0].City

	commonTown := locations[0].Town
	for _, loc := range locations {
		// 県名と市名が異なる場合は共通部分はない
		if loc.Prefecture != prefecture || loc.City != city {
			return ""
		}

		// 町名の共通部分を取得
		town := loc.Town
		if !strings.HasPrefix(town, commonTown) && !strings.HasPrefix(commonTown, town) {

			townParts := strings.Split(town, "")
			commonTownParts := strings.Split(commonTown, "")
			commonParts := []string{}

			for i, char := range townParts {
				if i < len(commonTownParts) && string(commonTownParts[i]) == char {
					commonParts = append(commonParts, char)
				} else {
					break
				}
			}
			commonTown = strings.Join(commonParts, "")

		} else if len(town) < len(commonTown) {
			commonTown = town
		}
	}
	
	// 共通部分の住所を返す
	return prefecture + city + commonTown
}

// 緯度差・経度差から東京駅までの距離の計算
func calculateDistance(x, y float64) float64 {
	xt := TokyoStationX
	yt := TokyoStationY
	R  := EarthRadius
	
	dX := (x - xt) * math.Cos(math.Pi*(y+yt)/360)
	dY := y - yt
	
	distance := (math.Pi * R / 180) * math.Sqrt(dX*dX + dY*dY)
	
	// ⼩数点第⼀位まで表⽰(四捨五⼊)
	return math.Round(distance*10) / 10
}

// データベース接続
var db *sql.DB

// アクセスログを保存
func saveAccessLog(postalCode string) error {
	_, err := db.Exec("INSERT INTO access_logs (postal_code) VALUES (?)", postalCode)
	return err
}

func addressHandler(w http.ResponseWriter, r *http.Request) {
	// リクエストメソッドの確認
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// 郵便番号の取得
	postalCode := r.URL.Query().Get("postal_code")
	
	// 郵便番号7桁か検証
	match, _ := regexp.MatchString("^[0-9]{7}$", postalCode)
	if !match {
		http.Error(w, "Invalid postal code format. Must be 7 digits.", http.StatusBadRequest)
		return
	}
	
	// 外部APIへリクエスト
	externalAPIURL := fmt.Sprintf("https://geoapi.heartrails.com/api/json?method=searchByPostal&postal=%s", postalCode)
	resp, err := http.Get(externalAPIURL)
	if err != nil {
		http.Error(w, "Failed to fetch data from external API", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	
	// レスポンスの読み取り
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read response body", http.StatusInternalServerError)
		return
	}
	
	// JSONのパース
	var externalResponse ExternalAPIResponse
	if err := json.Unmarshal(body, &externalResponse); err != nil {
		http.Error(w, "Failed to parse external API response", http.StatusInternalServerError)
		return
	}
	
	// レスポンスの検証
	locations := externalResponse.Response.Location
	if len(locations) == 0 {
		http.Error(w, "No location found for the given postal code", http.StatusNotFound)
		return
	}

	// 各住所のうち、共通部分の住所を取得
	commonAddress := getCommonAddress(locations)
	
	// 各住所から東京駅までの距離の計算し、最大値を取得
	var maxDistance float64 = 0
	for _, loc := range locations {
		x, _ := strconv.ParseFloat(loc.X, 64)
		y, _ := strconv.ParseFloat(loc.Y, 64)
		
		distance := calculateDistance(x, y)
		if distance > maxDistance {
			maxDistance = distance
		}
	}
	
	// アクセスログを保存
	if err := saveAccessLog(postalCode); err != nil {
		log.Printf("Failed to save access log: %v", err)
	}
	
	// レスポンスの作成
	response := AddressResponse{
		PostalCode:       postalCode,
		HitCount:         len(locations),
		Address:          commonAddress,
		TokyoStaDistance: maxDistance,
	}
	
	// JSONに変換してレスポンスを返す
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func accessLogsHandler(w http.ResponseWriter, r *http.Request) {
	// リクエストメソッドの確認
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// アクセスログの集計
	rows, err := db.Query(`
		SELECT postal_code, COUNT(*) as request_count 
		FROM access_logs 
		GROUP BY postal_code 
		ORDER BY request_count DESC
	`)
	if err != nil {
		http.Error(w, "Failed to query access logs", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	// 結果の処理
	var accessLogs []AccessLogItem
	for rows.Next() {
		var item AccessLogItem
		if err := rows.Scan(&item.PostalCode, &item.RequestCount); err != nil {
			http.Error(w, "Failed to scan row", http.StatusInternalServerError)
			return
		}
		accessLogs = append(accessLogs, item)
	}
	
	// レスポンスの作成
	response := AccessLogsResponse{
		AccessLogs: accessLogs,
	}
	
	// JSONに変換してレスポンスを返す
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	// 環境変数からDBホストを取得
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}

	// MySQL接続情報を設定
	dsn := fmt.Sprintf("postal_api_db_user:postal_api_db_user_password@tcp(%s:3306)/postal_api_db", dbHost)

	// データベース接続
	var err error
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	
	// 接続テスト
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	
	fmt.Println("Connected to database successfully")
	
	// ルートハンドラーを設定
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Welcome to Postal API!")
	})
	
	// 郵便番号から該当する住所を取得するAPIエンドポイント
	http.HandleFunc("/address", addressHandler)
	
	// アクセスログを取得するAPIエンドポイント
	http.HandleFunc("/address/access_logs", accessLogsHandler)
	
	// サーバーを起動する
	fmt.Println("Starting server at port 8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}