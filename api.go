package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	jwt "github.com/AlvaroZev/rimont-messager/auth"
	"github.com/AlvaroZev/rimont-messager/config"
	whats "github.com/AlvaroZev/rimont-messager/connection"
	dbtypes "github.com/AlvaroZev/rimont-messager/dbtypes"
	nestdb "github.com/AlvaroZev/rimont-messager/sql"
	"github.com/google/uuid"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
)

var MainClientLoggedIn bool = false
var runOnce bool = false
var QrString string
var QrRunning bool

func LeadHandler(clients map[string]*whatsmeow.Client, whatsContainer *sqlstore.Container, db *sql.DB, secretKey []uint8, source string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		tokenString := r.Header.Get("Authorization")
		if tokenString == "" {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, "Missing authorization header")
			return
		}
		tokenString = tokenString[len("Bearer "):]
		err := jwt.VerifyToken(tokenString, secretKey)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, "Invalid token")
			return
		}

		var msgData struct {
			// Define the structure of your expected request body
			// For example, a message and recipient number
			Lead     dbtypes.Lead     `json:"lead"`
			Operator dbtypes.Operator `json:"operator"`
		}

		if err := json.NewDecoder(r.Body).Decode(&msgData); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var result = whats.LeadMessage(msgData.Lead, msgData.Operator, clients, whatsContainer, db, source)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}

func getChatHandler(db *sql.DB) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
			return
		}

		var msgData struct {
			// Define the structure of your expected request body
			// For example, a message and recipient number
			Conversation dbtypes.Conversation `json:"conversation"`
		}

		if err := json.NewDecoder(r.Body).Decode(&msgData); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Obtener mensajes de la base de datos
		fmt.Println(msgData.Conversation.ConversationID)
		messages, err := nestdb.GetChat(db, msgData.Conversation.ConversationID)
		fmt.Println(messages)
		if err != nil {
			log.Printf("Error obteniendo mensajes: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Responder con la lista de mensajes en formato JSON
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(messages)
	}
}

func LoginHandler(config config.Config, secretKey []uint8) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		if r.Method != http.MethodPost {
			http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
			return
		}

		var msgData struct {
			// Define the structure of your expected request body user and password
			Username string `json:"username"`
			Password string `json:"password"`
		}

		if err := json.NewDecoder(r.Body).Decode(&msgData); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if msgData.Username == config.JwtUser && msgData.Password == config.JwtPassword {
			w.Header().Set("Content-Type", "application/json")
			tokenString, err := jwt.CreateToken(msgData.Username, secretKey)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			//fmt.Fprint(w, tokenString)
			json.NewEncoder(w).Encode(tokenString)
			return
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintln(w, "Invalid credentials")
		}

	}
}

// this endpoint returns the qr code for the main client and the boolean MainClientLoggedIn
func QRHandler(qrstring *string, qrrun *bool, config *config.Config, clients map[string]*whatsmeow.Client, whatsContainer *sqlstore.Container, db *sql.DB, secretKey []uint8) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("QR HANDLER RUNNING")
		if r.Method != http.MethodGet {
			http.Error(w, "Only GET method is allowed", http.StatusMethodNotAllowed)
			return
		}

		//store the client
		client := clients[config.MainNumber]

		//get qr code
		if *qrrun == false && !client.IsLoggedIn() {
			go whats.EndpointQRGeneration(config.MainNumber, clients, whatsContainer, qrstring, qrrun)
			//sleep 3s
			time.Sleep(3 * time.Second)

		}

		MainClientLoggedIn = client.IsLoggedIn()
		if MainClientLoggedIn == true {
			fmt.Println("Main client logged in")
			*qrstring = ""
		}

		//respond with qr code and bool
		w.Header().Set("Content-Type", "application/json")

		var msgData struct {
			QrString           string `json:"qr"`
			MainClientLoggedIn bool   `json:"loggedin"`
		}

		//add endpoints
		if MainClientLoggedIn == true && runOnce == false {
			clients[config.MainNumber] = client
			InitEndpoints(config, clients, whatsContainer, db, secretKey)
			runOnce = true
		}

		msgData.QrString = *qrstring
		msgData.MainClientLoggedIn = MainClientLoggedIn

		json.NewEncoder(w).Encode(msgData)
	}
}

func main() {
	//var Core *whats.CoreType
	var err error

	config, err := config.NewLoadedConfig()
	if err != nil {
		panic(err)
	}

	//secret key for jwt
	var secretKey = []byte(config.JwtSecret)

	databaseURL := config.DatabaseURL
	if databaseURL == "" {
		fmt.Println("DATABASE_URL no está definida")
		return
	}
	// Inicializar la conexión a la base de datos
	db, err := nestdb.InitDB(databaseURL)
	if err != nil {
		fmt.Printf("Error al conectar a la base de datos: %v\n", err)
		return
	}
	defer db.Close()

	// Eliminar las tablas si existen
	// Esto limpiara la base de datos cada vez que se inicie el servidor.
	if config.RestartDBonInit {
		err = nestdb.DropTables(db)
		if err != nil {
			fmt.Printf("Error al eliminar las tablas: %v\n", err)
			return
		}

	}

	// Crear las tablas
	err = nestdb.CreateTables(db)
	if err != nil {
		fmt.Printf("Error al crear las tablas: %v\n", err)
		return
	}

	fmt.Println("Tablas creadas con éxito")

	//check if db is empty
	_, err = nestdb.GetMessage(db, uuid.Nil)

	//Llenar la base de datos con  datos de uuid.Nil, solo si se ha reiniciado la db o esta vacia
	if config.RestartDBonInit || err != nil {
		err = nestdb.AddLead(db, dbtypes.Lead{
			LeadID: uuid.Nil,
		})
		if err != nil {
			fmt.Printf("Error al añadir el lead: %v\n", err)
			return
		}
		err = nestdb.AddOperator(db, dbtypes.Operator{
			OperatorID: uuid.Nil,
		})
		if err != nil {
			fmt.Printf("Error al añadir el operador: %v\n", err)
			return
		}
		err = nestdb.AddConversation(db, dbtypes.Conversation{
			ConversationID: uuid.Nil,
		})

		err = nestdb.AddMessage(db, dbtypes.Message{
			MessageID:           uuid.Nil,
			MessageLeadID:       uuid.Nil,
			MessageOperatorID:   uuid.Nil,
			ConversationID:      uuid.Nil,
			MessageCreatedAt:    time.Now().Format(time.RFC3339),
			MessageWspTimestamp: time.Now().Format(time.RFC3339),
		})
		if err != nil {
			fmt.Printf("Error al añadir el mensaje: %v\n", err)
			return
		}
	}

	// añadir main whatsapp number como operador principal
	MainOperatorID, err := uuid.Parse("00000000-0000-0000-0000-000000000001")
	if err != nil {
		fmt.Printf("Error al parsear el UUID del operador principal: %v\n", err)
	}

	_, errGetOperator := nestdb.GetOperator(db, MainOperatorID)
	if errGetOperator != nil {
		fmt.Printf("Error al obtener el operador principal, no existe: %v\n", errGetOperator)
		//añadir nuevo operador
		err = nestdb.AddOperator(db, dbtypes.Operator{
			OperatorID:            MainOperatorID,
			OperatorName:          "a",
			OperatorSurnames:      "a",
			OperatorPhone:         config.MainNumber,
			OperatorBackOfficeID:  "0",
			OperatorDevice:        "0",
			OperatorPersonalPhone: "0",
		})
		if err != nil {
			fmt.Printf("Error al añadir el operador principal: %v\n", err)

		}

	}

	err = nestdb.UpdateOperator(db, MainOperatorID, dbtypes.Operator{
		OperatorID:            MainOperatorID,
		OperatorName:          "a",
		OperatorSurnames:      "a",
		OperatorPhone:         config.MainNumber,
		OperatorBackOfficeID:  "0",
		OperatorDevice:        "0",
		OperatorPersonalPhone: "0",
	})
	if err != nil {
		fmt.Printf("Error al actualizar el operador principal: %v\n", err)
		return
	}

	dbMainOperator, err := nestdb.GetOperator(db, MainOperatorID)
	if err != nil {
		fmt.Printf("Error al obtener el operador principal: %v\n", err)
		return
	}

	// Crear un nuevo map de clientes de WhatsApp
	clients := map[string]*whatsmeow.Client{}

	//unique
	whatsContainer := whats.NewWhatsAppContainer(config.WhatsAppDatabaseName)

	//get main operator client

	mainClient, err := whats.GetClientSessionFromContainerJID(whatsContainer, dbMainOperator)
	if err != nil {
		fmt.Println("Error getting session with only number:", err)
		return
	}

	QrRunning = false
	QrString = ""
	//store client
	clients[config.MainNumber] = mainClient

	//add main client export qr endpoint
	http.HandleFunc("/qr", QRHandler(&QrString, &QrRunning, config, clients, whatsContainer, db, secretKey))
	fmt.Println("QR Endpoint is running")
	//response 200 on /
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	go whats.EndpointQRGeneration(config.MainNumber, clients, whatsContainer, &QrString, &QrRunning)

	//Sleep 10s to wait for qr code to be generated
	time.Sleep(10 * time.Second)
	MainClientLoggedIn = mainClient.IsLoggedIn()

	if MainClientLoggedIn == true && runOnce == false {
		clients[config.MainNumber] = mainClient
		InitEndpoints(config, clients, whatsContainer, db, secretKey)
		runOnce = true
	}

	log.Fatal(http.ListenAndServe(":8080", nil))

	// Listen to Ctrl+C (you can also do something else that prevents the program from exiting)
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	for _, client := range clients {
		client.Disconnect()
	}

}

func InitEndpoints(config *config.Config, clients map[string]*whatsmeow.Client, whatsContainer *sqlstore.Container, db *sql.DB, secretKey []uint8) {
	http.HandleFunc("/login", LoginHandler(*config, secretKey))
	http.HandleFunc("/calllead", LeadHandler(clients, whatsContainer, db, secretKey, "call"))
	http.HandleFunc("/noanswerlead", LeadHandler(clients, whatsContainer, db, secretKey, "noanswer"))
	http.HandleFunc("/feedbacklead", LeadHandler(clients, whatsContainer, db, secretKey, "feedback"))
	http.HandleFunc("/conversation", getChatHandler(db))

	fmt.Println("Server is running on port :8080")
}
