package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
	"log"
	"net/http"
	"os"
	"strings"
)

type Motor struct {
	SerialNo        string `json:"serial_no"`
	MotorModel      string `json:"motor_model"`
	RPM             int    `json:"rpm"`
	Phase           string `json:"phase"`
	PartyName       string `json:"party_name"`
	DispatchDate    string `json:"dispatch_date"`
	TransportAgency string `json:"transport_agency"`
	LREwayBill      string `json:"lr_eway_bill"`
	TestCertificate string `json:"test_certificate"`
	PartyAddress    string `json:"party_address"`
	HPKW            string `json:"hp_kw"`
	Remarks         string `json:"remarks"`
}

var db *sql.DB

func initDB() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"))

	var errDB error
	db, errDB = sql.Open("postgres", connStr)
	if errDB != nil {
		log.Fatal(errDB)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	fmt.Println("Connected to PostgreSQL")
}

func registerMotor(w http.ResponseWriter, r *http.Request) {
	var motor Motor
	err := json.NewDecoder(r.Body).Decode(&motor)
	if err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	query := `INSERT INTO motors (serial_no, motor_model, rpm, phase, party_name, dispatch_date, 
              transport_agency, lr_or_eway_bill, test_certificate, party_address, hp_kw, remarks) 
              VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	_, err = db.Exec(query, motor.SerialNo, motor.MotorModel, motor.RPM, motor.Phase, motor.PartyName,
		motor.DispatchDate, motor.TransportAgency, motor.LREwayBill, motor.TestCertificate,
		motor.PartyAddress, motor.HPKW, motor.Remarks)

	if err != nil {
		http.Error(w, "Error inserting data", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "Motor registered successfully"})
}

func fetchMotor(w http.ResponseWriter, r *http.Request) {
	// Get query params
	serial := r.URL.Query().Get("serial_no")
	serial = strings.TrimSpace(serial)
	party := r.URL.Query().Get("party_name")
	party = strings.TrimSpace(party)

	// Prepare SQL query based on available parameters
	var rows *sql.Rows
	var err error

	// Handle queries with variable names
	var query string
	if serial != "" && party != "" {
		query = "SELECT * FROM motors WHERE serial_no = $1 AND party_name = $2"
		rows, err = db.Query(query, serial, party)
	} else if party == "" {
		query = "SELECT * FROM motors WHERE serial_no = $1"
		rows, err = db.Query(query, serial)
	} else if serial == "" {
		query = "SELECT * FROM motors WHERE party_name = $1"
		rows, err = db.Query(query, party)
	} else {
		http.Error(w, "No valid query parameters provided", http.StatusBadRequest)
		return
	}

	// Query the database
	if err != nil {
		http.Error(w, "Error fetching motors: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var motors []map[string]interface{}

	// Iterate over the rows and append results
	for rows.Next() {
		var motor Motor
		err := rows.Scan(&motor.SerialNo, &motor.MotorModel, &motor.RPM, &motor.Phase, &motor.PartyName,
			&motor.DispatchDate, &motor.TransportAgency, &motor.LREwayBill, &motor.TestCertificate,
			&motor.PartyAddress, &motor.HPKW, &motor.Remarks)
		if err != nil {
			http.Error(w, "Error scanning row: "+err.Error(), http.StatusInternalServerError)
			return
		}
		motors = append(motors, map[string]interface{}{
			"serial_no":        motor.SerialNo,
			"motor_model":      motor.MotorModel,
			"rpm":              motor.RPM,
			"phase":            motor.Phase,
			"party_name":       motor.PartyName,
			"dispatch_date":    motor.DispatchDate,
			"transport_agency": motor.TransportAgency,
			"lr_eway_bill":     motor.LREwayBill,
			"test_certificate": motor.TestCertificate,
			"party_address":    motor.PartyAddress,
			"hp_kw":            motor.HPKW,
			"remarks":          motor.Remarks,
		})
	}

	// Handle no results found
	if len(motors) == 0 {
		http.Error(w, "No motors found", http.StatusNotFound)
		return
	}

	// Return the results as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(motors)
}

func main() {
	initDB()
	r := mux.NewRouter()

	r.HandleFunc("/fetch", fetchMotor).Methods("GET")
	r.HandleFunc("/register", registerMotor).Methods("POST")

	// Enable CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"}, // React frontend URL
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	})

	handler := c.Handler(r)
	log.Println("Server running on :8080")
	http.ListenAndServe(":8080", handler)
}
