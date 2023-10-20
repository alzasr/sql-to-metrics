package main

import (
	"github.com/alzasr/sql-to-metrics/internal"
	_ "github.com/jackc/pgx/v5/stdlib"
	"gopkg.in/yaml.v2"
	"log"
	"os"
)

func main() {
	settingsFile, err := os.ReadFile("jobs.yaml")
	if err != nil {
		log.Fatal(err)
	}
	settings := &internal.Settings{}
	err = yaml.Unmarshal(settingsFile, settings)
	if err != nil {
		log.Fatal(err)
	}
	err = internal.Run(settings)
	if err != nil {
		log.Fatal(err)
	}
}
