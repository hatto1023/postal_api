CREATE DATABASE IF NOT EXISTS postal_api_db;
USE postal_api_db;

CREATE TABLE IF NOT EXISTS `access_logs` (
    `id` INT AUTO_INCREMENT NOT NULL,
    `postal_code` VARCHAR(8) NOT NULL,
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`)
);

CREATE USER 'postal_api_db_user'@'%' IDENTIFIED BY 'postal_api_db_user_password';
GRANT ALL PRIVILEGES ON postal_api_db.* TO 'postal_api_db_user'@'%';
FLUSH PRIVILEGES;