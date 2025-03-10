package nestdb

import (
	"database/sql"

	"github.com/google/uuid"

	dbtypes "github.com/AlvaroZev/rimont-messager/dbtypes"
	_ "github.com/lib/pq"
)

// Función para inicializar la conexión a la base de datos
func InitDB(dataSourceName string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dataSourceName)
	if err != nil {
		return nil, err
	}
	return db, nil
}

// Funciones para crear las tablas
func CreateTables(db *sql.DB) error {
	// Crear tabla Leads
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS Leads (
        LeadID UUID PRIMARY KEY,
        LeadName TEXT,
        LeadSurnames TEXT,
        LeadPhone TEXT,
        LeadInterestBrand TEXT,
        LeadInterestModel TEXT,
		LeadProvince TEXT,
		LeadDealership TEXT,
		LeadBackOfficeID TEXT
    )`)
	if err != nil {
		return err
	}

	// Crear tabla Operators
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS Operators (
        OperatorID UUID PRIMARY KEY,
        OperatorName TEXT,
        OperatorSurnames TEXT,
        OperatorPhone TEXT,
		OperatorBackOfficeID TEXT,
		OperatorDevice TEXT,
		OperatorPersonalPhone TEXT
    )`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS Conversations (
        ConversationID UUID PRIMARY KEY,
        LeadID UUID,
        OperatorID UUID,
		FOREIGN KEY (LeadID) REFERENCES Leads (LeadID),
        FOREIGN KEY (OperatorID) REFERENCES Operators (OperatorID)
    )`)
	if err != nil {
		return err
	}

	// Crear tabla Messages
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS Messages (
        MessageID UUID PRIMARY KEY,
		MessageText TEXT,
        MessageCreatedAt TIMESTAMP,
        MessageWspTimestamp TIMESTAMP,
        MessageParentID UUID,
        MessageLeadID UUID,
        MessageOperatorID UUID,
		ConversationID UUID,
		MessageWspID TEXT,
		FOREIGN KEY (MessageParentID) REFERENCES Messages (MessageID),
        FOREIGN KEY (MessageLeadID) REFERENCES Leads (LeadID),
        FOREIGN KEY (MessageOperatorID) REFERENCES Operators (OperatorID),
		FOREIGN KEY (ConversationID) REFERENCES Conversations (ConversationID)
    )`)
	if err != nil {
		return err
	}

	//crear tabla de handlers
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS Handlers (
		HandlerID INT,
		ConversationID UUID PRIMARY KEY,
		FOREIGN KEY (ConversationID) REFERENCES Conversations (ConversationID)
	)`)
	if err != nil {
		return err
	}

	return nil
}

// Función para eliminar las tablas si existen
func DropTables(db *sql.DB) error {
	// Eliminar tabla Messages
	_, err := db.Exec("DROP TABLE IF EXISTS Messages")
	if err != nil {
		return err
	}

	// Eliminar tabla Handlers
	_, err = db.Exec("DROP TABLE IF EXISTS Handlers")
	if err != nil {
		return err
	}

	// Eliminar tabla Conversations
	_, err = db.Exec("DROP TABLE IF EXISTS Conversations")
	if err != nil {
		return err
	}

	// Eliminar tabla Operators
	_, err = db.Exec("DROP TABLE IF EXISTS Operators")
	if err != nil {
		return err
	}

	// Eliminar tabla Leads
	_, err = db.Exec("DROP TABLE IF EXISTS Leads")
	if err != nil {
		return err
	}

	return nil
}

// Funciones para añadir elementos a las tablas
func AddLead(db *sql.DB, lead dbtypes.Lead) error {
	_, err := db.Exec(`INSERT INTO Leads (LeadID, LeadName, LeadSurnames, LeadPhone, LeadInterestBrand, LeadInterestModel, LeadProvince, LeadDealership, LeadBackOfficeID) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`, lead.LeadID, lead.LeadName, lead.LeadSurnames, lead.LeadPhone, lead.LeadInterestBrand, lead.LeadInterestModel, lead.LeadProvince, lead.LeadDealership, lead.LeadBackOfficeID)
	return err
}

func AddOperator(db *sql.DB, operator dbtypes.Operator) error {
	_, err := db.Exec(`INSERT INTO Operators (OperatorID, OperatorName, OperatorSurnames, OperatorPhone, OperatorBackOfficeID, OperatorDevice, OperatorPersonalPhone) VALUES ($1, $2, $3, $4, $5, $6, $7)`, operator.OperatorID, operator.OperatorName, operator.OperatorSurnames, operator.OperatorPhone, operator.OperatorBackOfficeID, operator.OperatorDevice, operator.OperatorPersonalPhone)
	return err
}

func AddMessage(db *sql.DB, message dbtypes.Message) error {
	_, err := db.Exec(`INSERT INTO Messages (MessageID, MessageText, MessageCreatedAt, MessageWspTimestamp, MessageParentID, MessageLeadID, MessageOperatorID, ConversationID, MessageWspID) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`, message.MessageID, message.MessageText, message.MessageCreatedAt, message.MessageWspTimestamp, message.MessageParentID, message.MessageLeadID, message.MessageOperatorID, message.ConversationID, message.MessageWspID)
	return err
}

func AddConversation(db *sql.DB, conversation dbtypes.Conversation) error {
	_, err := db.Exec(`INSERT INTO Conversations (ConversationID, LeadID, OperatorID) VALUES ($1, $2, $3)`, conversation.ConversationID, conversation.LeadID, conversation.OperatorID)
	return err
}

func AddHandler(db *sql.DB, HandlerID uint32, ConversationID uuid.UUID) error {
	_, err := db.Exec(`INSERT INTO Handlers (HandlerID, ConversationID) VALUES ($1, $2)`, HandlerID, ConversationID)
	return err
}

func DeleteHandler(db *sql.DB, ConversationID uuid.UUID) error {
	_, err := db.Exec(`DELETE FROM Handlers WHERE ConversationID = $1`, ConversationID)
	return err
}

func UpdateOperatorDevice(db *sql.DB, operatorID uuid.UUID, newDeviceValue string) error {
	// Prepare the SQL statement
	stmt := `UPDATE Operators SET OperatorDevice = $1 WHERE OperatorID = $2`

	// Execute the SQL statement
	_, err := db.Exec(stmt, newDeviceValue, operatorID)
	return err
}

func UpdateOperator(db *sql.DB, operatorID uuid.UUID, newOperatorValue dbtypes.Operator) error {
	// Prepare the SQL statement
	stmt := `UPDATE Operators SET OperatorName = $1, OperatorSurnames = $2, OperatorPhone = $3, OperatorBackOfficeID = $4, OperatorDevice = $5, OperatorPersonalPhone = $6 WHERE OperatorID = $7`

	// Execute the SQL statement
	_, err := db.Exec(stmt, newOperatorValue.OperatorName, newOperatorValue.OperatorSurnames, newOperatorValue.OperatorPhone, newOperatorValue.OperatorBackOfficeID, newOperatorValue.OperatorDevice, newOperatorValue.OperatorPersonalPhone, operatorID)
	return err
}

func UpdateLead(db *sql.DB, leadID uuid.UUID, newLeadValue dbtypes.Lead) error {
	// Prepare the SQL statement
	stmt := `UPDATE Leads SET LeadName = $1, LeadSurnames = $2, LeadPhone = $3, LeadInterestBrand = $4, LeadInterestModel = $5, LeadProvince = $6, LeadDealership = $7, LeadBackOfficeID = $8 WHERE LeadID = $9`

	// Execute the SQL statement
	_, err := db.Exec(stmt, newLeadValue.LeadName, newLeadValue.LeadSurnames, newLeadValue.LeadPhone, newLeadValue.LeadInterestBrand, newLeadValue.LeadInterestModel, newLeadValue.LeadProvince, newLeadValue.LeadDealership, newLeadValue.LeadBackOfficeID, leadID)
	return err
}

// Funciones para leer
func GetLead(db *sql.DB, leadID uuid.UUID) (dbtypes.Lead, error) {
	var lead dbtypes.Lead
	err := db.QueryRow("SELECT LeadID, LeadName, LeadSurnames, LeadPhone, LeadInterestBrand, LeadInterestModel, LeadProvince, LeadDealership, LeadBackOfficeID FROM Leads WHERE LeadID = $1", leadID).Scan(&lead.LeadID, &lead.LeadName, &lead.LeadSurnames, &lead.LeadPhone, &lead.LeadInterestBrand, &lead.LeadInterestModel, &lead.LeadProvince, &lead.LeadDealership, &lead.LeadBackOfficeID)
	if err != nil {
		return lead, err
	}
	return lead, nil
}

func GetLeadByBackOfficeID(db *sql.DB, leadBackOfficeID string) (dbtypes.Lead, error) {
	var lead dbtypes.Lead
	err := db.QueryRow("SELECT LeadID, LeadName, LeadSurnames, LeadPhone, LeadInterestBrand, LeadInterestModel, LeadProvince, LeadDealership, LeadBackOfficeID FROM Leads WHERE LeadBackOfficeID = $1", leadBackOfficeID).Scan(&lead.LeadID, &lead.LeadName, &lead.LeadSurnames, &lead.LeadPhone, &lead.LeadInterestBrand, &lead.LeadInterestModel, &lead.LeadProvince, &lead.LeadDealership, &lead.LeadBackOfficeID)
	if err != nil {
		return lead, err
	}
	return lead, nil
}

func GetOperator(db *sql.DB, operatorID uuid.UUID) (dbtypes.Operator, error) {
	var operator dbtypes.Operator
	err := db.QueryRow("SELECT OperatorID, OperatorName, OperatorSurnames, OperatorPhone, OperatorBackOfficeID, OperatorDevice, OperatorPersonalPhone FROM Operators WHERE OperatorID = $1", operatorID).Scan(&operator.OperatorID, &operator.OperatorName, &operator.OperatorSurnames, &operator.OperatorPhone, &operator.OperatorBackOfficeID, &operator.OperatorDevice, &operator.OperatorPersonalPhone)
	if err != nil {
		return operator, err
	}
	return operator, nil
}

func GetOperatorByBackOfficeID(db *sql.DB, operatorBackOfficeID string) (dbtypes.Operator, error) {
	var operator dbtypes.Operator
	err := db.QueryRow("SELECT OperatorID, OperatorName, OperatorSurnames, OperatorPhone, OperatorBackOfficeID, OperatorDevice, OperatorPersonalPhone FROM Operators WHERE OperatorBackOfficeID = $1", operatorBackOfficeID).Scan(&operator.OperatorID, &operator.OperatorName, &operator.OperatorSurnames, &operator.OperatorPhone, &operator.OperatorBackOfficeID, &operator.OperatorDevice, &operator.OperatorPersonalPhone)
	if err != nil {
		return operator, err
	}
	return operator, nil
}

func GetMessage(db *sql.DB, messageID uuid.UUID) (dbtypes.Message, error) {
	var message dbtypes.Message
	err := db.QueryRow("SELECT MessageID, MessageText, MessageCreatedAt, MessageWspTimestamp, MessageParentID, MessageLeadID, MessageOperatorID, ConversationID, MessageWspID FROM Messages WHERE MessageID = $1", messageID).Scan(&message.MessageID, &message.MessageText, &message.MessageCreatedAt, &message.MessageWspTimestamp, &message.MessageParentID, &message.MessageLeadID, &message.MessageOperatorID, &message.ConversationID, &message.MessageWspID)
	if err != nil {
		return message, err
	}
	return message, nil
}

func GetConversation(db *sql.DB, conversationID uuid.UUID) (dbtypes.Conversation, error) {
	var conversation dbtypes.Conversation
	err := db.QueryRow("SELECT ConversationID, LeadID, OperatorID FROM Conversations WHERE ConversationID = $1", conversationID).Scan(&conversation.ConversationID, &conversation.LeadID, &conversation.OperatorID)
	if err != nil {
		return conversation, err
	}
	return conversation, nil
}

func GetHandler(db *sql.DB, ConversationID uuid.UUID) (uint32, error) {
	var handler uint32
	err := db.QueryRow("SELECT HandlerID FROM Handlers WHERE ConversationID = $1", ConversationID).Scan(&handler)
	if err != nil {
		return handler, err
	}
	return handler, nil
}

func GetConversationIdByParticipants(db *sql.DB, leadID uuid.UUID, operatorID uuid.UUID) (dbtypes.Conversation, error) {
	var conversation dbtypes.Conversation
	err := db.QueryRow("SELECT ConversationID, LeadID, OperatorID FROM Conversations WHERE LeadID = $1 AND OperatorID = $2", leadID, operatorID).Scan(&conversation.ConversationID, &conversation.LeadID, &conversation.OperatorID)
	if err != nil {
		return conversation, err
	}
	return conversation, nil

}

func GetChat(db *sql.DB, conversationID uuid.UUID) ([]dbtypes.Message, error) {
	var messages []dbtypes.Message

	rows, err := db.Query("SELECT MessageID, MessageText, MessageCreatedAt, MessageWspTimestamp, MessageParentID, MessageLeadID, MessageOperatorID, ConversationID, MessageWspID FROM Messages WHERE ConversationID = $1 ORDER BY MessageCreatedAt ASC", conversationID)
	if err != nil {
		return messages, err
	}

	defer rows.Close()

	for rows.Next() {
		var message dbtypes.Message
		err := rows.Scan(&message.MessageID, &message.MessageText, &message.MessageCreatedAt, &message.MessageWspTimestamp, &message.MessageParentID, &message.MessageLeadID, &message.MessageOperatorID, &message.ConversationID, &message.MessageWspID)
		if err != nil {
			return messages, err
		}
		messages = append(messages, message)
	}

	return messages, nil
}

func GetConversationLastMessage(db *sql.DB, conversationID uuid.UUID) (dbtypes.Message, error) {
	var message dbtypes.Message
	err := db.QueryRow("SELECT MessageID, MessageText, MessageCreatedAt, MessageWspTimestamp, MessageParentID, MessageLeadID, MessageOperatorID, ConversationID, MessageWspID FROM Messages WHERE ConversationID = $1 ORDER BY MessageCreatedAt DESC LIMIT 1", conversationID).Scan(&message.MessageID, &message.MessageText, &message.MessageCreatedAt, &message.MessageWspTimestamp, &message.MessageParentID, &message.MessageLeadID, &message.MessageOperatorID, &message.ConversationID, &message.MessageWspID)
	if err != nil {
		return message, err
	}
	return message, nil
}
