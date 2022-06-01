package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/spf13/viper"
)

var (
	db  *sql.DB
	err error
)

func main() {
	fmt.Println("Starting Soil Sensor API")

	// initialize config file
	viper.SetConfigType("yaml")
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.SetDefault("dbusername", "123")
	viper.SetDefault("dbpassword", "456")
	viper.SetDefault("dbaddress", "localhost")
	viper.SetDefault("dbport", 3306)
	viper.SetDefault("dbname", "database1")
	viper.SetDefault("dbtable", "table1")
	viper.SetDefault("httpport", 15000)
	viper.SetDefault("espkey", uuid.New().String())

	// reading config file
	fmt.Println("Reading config file")
	if err = viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			fmt.Println("Config file is not found, creating new one.")
			err = viper.SafeWriteConfig()
			if err != nil {
				fmt.Printf("Creating config file error: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("Please modify setting in the file and relaunch the program")
			os.Exit(1)
		} else {
			fmt.Printf("Reading config file error: %v\n", err)
		}
	}

	// get variables from config
	username := viper.GetString("dbusername")
	password := viper.GetString("dbpassword")
	address := viper.GetString("dbaddress")
	port := viper.GetInt("dbport")
	dbName := viper.GetString("dbname")

	// connect to database
	fmt.Printf("Establishing connection to MySQL server: %s:%d\n", address, port)
	db, err = sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8", username, password, address, port, dbName))
	if err != nil {
		fmt.Printf("Establishing to mysql server error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("MySQL server successfully connected")

	fmt.Println("Started http api server")
	// setup http server
	http.HandleFunc("/create", handleDataReceivedRequest)
	// start listening http request
	err := http.ListenAndServe(fmt.Sprintf(":%d", viper.GetInt("httpport")), nil)
	if err != nil {
		fmt.Printf("Listening http request error: %v\n", err)
		os.Exit(1)
	}
}

// Data struct from ESPHome
type Data struct {
	Key         string  `json:"key"`
	Temperature float32 `json:"temperature"`
	Humidity    float32 `json:"humidity"`
	//Light       float32 `json:"light"`
	//Moisture    float32 `json:"moisture"`
}

// handleDataReceivedRequest handle http requests
func handleDataReceivedRequest(w http.ResponseWriter, r *http.Request) {
	// return if request type is not post
	if r.Method != "POST" {
		_, err := fmt.Fprintln(w, "API only can be called by POST Request")
		if err != nil {
			fmt.Printf("Writing to http client error: %v\n", err)
		}
		return
	}
	// read full request body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Printf("Reading http request body error: %v\n", err)
		return
	}
	// get data from request body
	data := Data{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		fmt.Printf("Unmarshal json from http request body error: %v\n", err)
		return
	}
	// check key is exist and match the configured one
	if data.Key != viper.GetString("espkey") {
		_, err := fmt.Fprintln(w, "Key is not provided or incorrect")
		if err != nil {
			fmt.Printf("Writing to http client error: %v\n", err)
		}
		return
	}
	// prepare mysql query
	stmt, err := db.Prepare("INSERT INTO " + viper.GetString("dbtable") + " (time,temperature,humidity,light,moisture) VALUES(?,?,?,?,?);")
	if err != nil {
		fmt.Printf("Preparing mysql query error: %v\n", err)
		return
	}
	// execute mysql query
	_, err = stmt.Exec(time.Now().Format("2006-01-02T15:04:05Z07:00"), data.Temperature, data.Humidity)
	if err != nil {
		fmt.Printf("Executing mysql query error: %v\n", err)
		return
	}
	fmt.Fprintln(w, "Success")
}
