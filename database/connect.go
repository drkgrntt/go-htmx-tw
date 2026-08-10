package database

import (
	"fmt"
	"log"

	"github.com/drkgrntt/htmx-test/utils"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

var db *sqlx.DB

func Connect() {
	config := utils.GetConfig()
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s", config.DbHost, config.DbUser, config.DbPassword, config.DbName, config.DbPort)
	connected, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		log.Fatalln(err)
	}
	db = connected

	Migrate()
}

func GetDatabase() *sqlx.DB {
	return db
}
