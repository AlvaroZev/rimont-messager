package whats

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/AlvaroZev/rimont-messager/config"
	"github.com/AlvaroZev/rimont-messager/dbtypes"
	nestdb "github.com/AlvaroZev/rimont-messager/sql"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal/v3"
	"github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

type CoreType struct {
	whatsClient *whatsmeow.Client
	db          *sql.DB
	config      *config.Config
}

func EndpointQRGeneration(MainNumber string, clients map[string]*whatsmeow.Client, whatsContainer *sqlstore.Container, qrstring *string, qrrun *bool) {
	//get client
	client := clients[MainNumber]
	//connect and verify log in
	if !client.IsConnected() {
		err := client.Connect()
		if err != nil {
			fmt.Println("Error in test connect mainClient:", err)
		}
		//Sleep 3s //TODO increase to 30s for production
		time.Sleep(3 * time.Second)
	}

	loggedIn := client.IsLoggedIn()
	//if it is not logged in, return qr code
	if !loggedIn {
		client.Disconnect()
		//Sleep 2s to wait for disconnection
		time.Sleep(2 * time.Second)

		//get a new client
		clients[MainNumber] = NewWhatsAppClientNewDevice(whatsContainer)
		//update client variable
		client = clients[MainNumber]

		qrChan, _ := client.GetQRChannel(context.Background())
		if !client.IsConnected() {
			err := client.Connect()
			if err != nil {
				panic(err)
			}
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				// Render the QR code here
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				// or just manually `echo 2@... | qrencode -t ansiutf8` in a terminal
				fmt.Println("Main QR code:", evt.Code)
				*qrstring = evt.Code
				*qrrun = true
			} else if evt.Event == "timeout" {
				*qrstring = ""
				*qrrun = false
			} else if evt.Event == "success" {
				*qrstring = ""
				*qrrun = false
				//sleep 3s
				time.Sleep(3 * time.Second)
			} else if evt.Event == "error" {

				*qrstring = ""
				*qrrun = false

				err := client.Logout()
				if err != nil {
					fmt.Println("Error in logout:", err)
					client.Disconnect()
					client.Store.Delete()
				}

				//get a new client
				clients[MainNumber] = NewWhatsAppClientNewDevice(whatsContainer)
				//update client variable
				client = clients[MainNumber]

				//call recursive or not?
			} else {
				fmt.Println("Main Login event:", evt.Event)
				*qrstring = ""
				*qrrun = false
			}
		}
	} else if loggedIn {
		//Already connected
		*qrrun = !client.IsLoggedIn()
		fmt.Println("Already connected")
	}

}

func QRConnection(client *whatsmeow.Client, clients map[string]*whatsmeow.Client, whatsContainer *sqlstore.Container, db *sql.DB, operator dbtypes.Operator) {

	if !client.IsConnected() {
		err := client.Connect()
		if err != nil {
			fmt.Println("Error in connect Client:", err)
		}
		//Sleep 3s //TODO increase to 30s for production
		time.Sleep(3 * time.Second)
	}

	MainOperatorID, err := uuid.Parse("00000000-0000-0000-0000-000000000001")
	if err != nil {
		fmt.Printf("Error al parsear el UUID del operador principal: %v\n", err)
	}

	//get dbMainOperator
	dbMainOperator, err := nestdb.GetOperator(db, MainOperatorID)
	if err != nil {
		fmt.Printf("Error al obtener el operador principal: %v\n", err)
		return
	}

	//get mainClient ready
	mainClient := clients[dbMainOperator.OperatorPhone]

	if mainClient == nil {
		//if not found, panic and stop operation
		panic("Main Operator not found")
	}

	if !mainClient.IsConnected() {
		err := mainClient.Connect()
		if err != nil {
			fmt.Println("Error in test connect mainClient:", err)
		}
		//Sleep 3s //TODO increase to 30s for production
		time.Sleep(3 * time.Second)

	}

	//get operator jid from phone
	operatorpersonaljid, err := types.ParseJID("51" + operator.OperatorPersonalPhone + "@s.whatsapp.net")
	if err != nil {
		fmt.Println("Error in parse JID for operator personal phone:", err)
		panic(err)
	}

	clientLoggedIn := client.IsLoggedIn()

	//up to this point, client should be connected and not logged in (the first time). and mainClient should be connected and logged in

	if !clientLoggedIn && mainClient.IsLoggedIn() {
		client.Disconnect()
		//Sleep 2s
		time.Sleep(2 * time.Second)

		//delete operator device locally
		operator.OperatorDevice = ""
		//get a new client in clients map
		clients[operator.OperatorPhone] = NewWhatsAppClientNewDevice(whatsContainer)
		//update client variable
		client = clients[operator.OperatorPhone]

		// No ID stored, new login
		qrChan, _ := client.GetQRChannel(context.Background())
		err := client.Connect()
		if err != nil {
			panic(err)
		}

		for evt := range qrChan {
			if evt.Event == "code" {
				// Render the QR code here
				//pick a  whatsapp client, from the env-var

				qr, err := qrcode.New(evt.Code, qrcode.Medium)
				if err != nil {
					fmt.Println("Error creating QR code:", err)
					return
				}

				//get qr to png format
				var png bytes.Buffer

				err = qr.Write(256, &png)
				if err != nil {
					fmt.Println("Error encoding QR code as PNG:", err)
					return
				}
				data := png.Bytes()

				uploaded, err := mainClient.Upload(context.Background(), data, whatsmeow.MediaImage)
				if err != nil {
					fmt.Println("Error in upload image:", err)
					panic(err)
				}
				msg := &waProto.Message{ImageMessage: &waProto.ImageMessage{
					Caption:       proto.String("Escanea el código QR para iniciar sesión"),
					Url:           proto.String(uploaded.URL),
					DirectPath:    proto.String(uploaded.DirectPath),
					MediaKey:      uploaded.MediaKey,
					Mimetype:      proto.String("image/png"),
					FileEncSha256: uploaded.FileEncSHA256,
					FileSha256:    uploaded.FileSHA256,
					FileLength:    proto.Uint64(uint64(len(data))),
				}}

				//send message
				_, err = mainClient.SendMessage(context.Background(), operatorpersonaljid, msg)
				if err != nil {
					fmt.Println("Error in send image Message:", err)
					panic(err)
				}

				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				// or just manually `echo 2@... | qrencode -t ansiutf8` in a terminal
				fmt.Println("QR code:", evt.Code)
			} else if evt.Event == "error" {
				//send a message to the operator to try again
				msg := &waProto.Message{Conversation: proto.String("Error al iniciar sesión, por favor, intenta de nuevo")}
				_, err = mainClient.SendMessage(context.Background(), operatorpersonaljid, msg)
				if err != nil {
					fmt.Println("Error in send retry Message:", err)
					panic(err)
				}

				err = client.Logout()
				if err != nil {
					fmt.Println("Error in logout:", err)
					client.Disconnect()
					client.Store.Delete()
				}

				//delete operator device locally
				operator.OperatorDevice = ""
				//get a new client in clients map
				clients[operator.OperatorPhone] = NewWhatsAppClientNewDevice(whatsContainer)
				//update client variable
				client = clients[operator.OperatorPhone]
				//non-signin clients are not stored (todo/tolearn)

				QRConnection(client, clients, whatsContainer, db, operator)
			} else {
				fmt.Println("Login event:", evt.Event)
			}
		}
	} else if client.IsConnected() && clientLoggedIn {
		//Already connected
		fmt.Println("Already logged-in")
	} else if !clientLoggedIn && !mainClient.IsLoggedIn() {
		fmt.Println("Main Operator not logged in, cannot send QR code")
		panic("Main Operator not logged in, cannot send QR code")

	}
}

// well documented general version of CallLeadMessage and MeetLeadMessage
func LeadMessage(lead dbtypes.Lead, operator dbtypes.Operator, clients map[string]*whatsmeow.Client, whatsContainer *sqlstore.Container, db *sql.DB, source string) dbtypes.Conversation {
	// 1. verify if the lead and operator exists in the database postgresql with backofficeID, if not create it

	dbLead, err := nestdb.GetLeadByBackOfficeID(db, lead.LeadBackOfficeID)
	if err != nil {
		fmt.Println("Error in get Lead:", err)
	}

	if dbLead.LeadID == uuid.Nil {
		//add uuid
		lead.LeadID = uuid.New()
		err := nestdb.AddLead(db, lead)
		if err != nil {
			fmt.Println("Error in add Lead:", err)
			panic(err)
		}
	} else {
		//update lead
		err := nestdb.UpdateLead(db, dbLead.LeadID, lead)
		if err != nil {
			fmt.Println("Error in update Lead:", err)
			panic(err)
		}
	}

	dbLead, err = nestdb.GetLeadByBackOfficeID(db, lead.LeadBackOfficeID)
	if err != nil {
		fmt.Println("Error in get Lead:", err)
	}

	dbOperator, err := nestdb.GetOperatorByBackOfficeID(db, operator.OperatorBackOfficeID)
	if err != nil {
		fmt.Println("Error in get Operator:", err)
	}

	if dbOperator.OperatorID == uuid.Nil {
		//add uuid
		operator.OperatorID = uuid.New()
		err := nestdb.AddOperator(db, operator)
		if err != nil {
			fmt.Println("Error in add Operator:", err)
			panic(err)
		}
	} else {
		//update operator
		//TODO: update only the device
		operator.OperatorDevice = dbOperator.OperatorDevice
		err := nestdb.UpdateOperator(db, dbOperator.OperatorID, operator)
		if err != nil {
			fmt.Println("Error in update Operator:", err)
			panic(err)
		}
	}

	dbOperator, err = nestdb.GetOperatorByBackOfficeID(db, operator.OperatorBackOfficeID)
	if err != nil {
		fmt.Println("Error in get Operator:", err)
	}

	// 2. verify if it exists a conversation between the lead and the operator, if not create it

	dbConversation, err := nestdb.GetConversationIdByParticipants(db, dbLead.LeadID, dbOperator.OperatorID)
	if err != nil {
		fmt.Println("Error in get Conversation:", err)
	}

	if dbConversation.ConversationID == uuid.Nil {
		//add uuid
		dbConversation.ConversationID = uuid.New()
		dbConversation.LeadID = dbLead.LeadID
		dbConversation.OperatorID = dbOperator.OperatorID
		fmt.Println(dbConversation.ConversationID)
		fmt.Println(dbConversation.LeadID)
		fmt.Println(dbConversation.OperatorID)
		err := nestdb.AddConversation(db, dbtypes.Conversation{
			ConversationID: dbConversation.ConversationID,
			LeadID:         dbConversation.LeadID,
			OperatorID:     dbConversation.OperatorID,
		})
		if err != nil {
			fmt.Println("Error in add Conversation:", err)
			panic(err)
		}
	}

	//3. get the whatsapp session of the specific operator

	//todo what if the phone from the http request is not the same as the one logged in? ask Jonathan

	//first find it inside clients map
	//if not found, create a new session
	client := clients[dbOperator.OperatorPhone]

	if clients[dbOperator.OperatorPhone] == nil {
		clients[dbOperator.OperatorPhone], err = GetClientSessionFromContainerADJID(whatsContainer, dbOperator)
		client = clients[dbOperator.OperatorPhone]
		//non-signin clients are not stored (todo/tolearn)
	}

	//connect to the account, sign-in
	QRConnection(client, clients, whatsContainer, db, dbOperator)

	client = clients[dbOperator.OperatorPhone] //actualizar cliente si es que se conectó con cliente nuevo
	//fmt.Println("Client device:", client.Store.ID.Device)
	//save the operator device in the db client.Store.ID
	dbOperator.OperatorDevice = strconv.Itoa(int(client.Store.ID.Device))
	err = nestdb.UpdateOperatorDevice(db, dbOperator.OperatorID, dbOperator.OperatorDevice)
	if err != nil {
		fmt.Println("Error in add Operator device:", err)
		panic(err)
	}

	// 4. add event handler

	//delete handler if exists
	delteHandlerId, err := nestdb.GetHandler(db, dbConversation.ConversationID)
	if err != nil {
		fmt.Println("Error in get Handler ID to delete:", err)
	}
	go client.RemoveEventHandler(delteHandlerId)
	//erase handler from the db
	err = nestdb.DeleteHandler(db, dbConversation.ConversationID)
	if err != nil {
		fmt.Println("Error in delete Handler at request :", err)
	}

	handlerId := client.AddEventHandler(LeadHandler(clients, whatsContainer, db, dbLead, dbOperator, dbConversation, source))

	//save eventhandler id to the db
	err = nestdb.AddHandler(db, handlerId, dbConversation.ConversationID)
	if err != nil {
		fmt.Println("Error in add Handler:", err)
		panic(err)
	}

	//this function runs only when triggered by operator. thus no previous conversation context is needed for the bot.
	//retrieve previous messages, get the last two messages, one from lead and one from operator

	// send the first message
	message, _, _, err := ConversationFlow("", "", dbLead, dbOperator, source)
	if err != nil {
		fmt.Println("Error in call Conversation Flow:", err)
		panic(err)
	}

	//generate lead JID, Peru Exclusive for the moment
	jid, err := types.ParseJID("51" + dbLead.LeadPhone + "@s.whatsapp.net")

	if err != nil {
		fmt.Println("Error in parse JID:", err)
		panic(err)
	}

	//send message
	msg := &waProto.Message{Conversation: proto.String(message)}
	response, err := client.SendMessage(context.Background(), types.JID(jid), msg)
	if err != nil {
		fmt.Println("Error in send Message:", err)
		panic(err)
	}

	//print message ID
	fmt.Println("Message ID:", response.ID)
	//print timestamp
	fmt.Println("Timestamp:", response)

	//save the message to the db

	err = nestdb.AddMessage(db, dbtypes.Message{
		MessageID:           uuid.New(),
		MessageText:         message,
		MessageCreatedAt:    time.Now().Format(time.RFC3339),
		MessageWspTimestamp: response.Timestamp.Format(time.RFC3339), //postgress timestamp format
		MessageParentID:     uuid.Nil,
		MessageLeadID:       uuid.Nil, //no id for the lead, it was sent from the operator
		MessageOperatorID:   dbOperator.OperatorID,
		ConversationID:      dbConversation.ConversationID,
		MessageWspID:        response.ID,
	})
	if err != nil {
		fmt.Println("Error in add Message:", err)
		panic(err)
	}

	//notify the backoffice api a new message with the timestamp
	//TODO

	return dbConversation
}

func NewWhatsAppContainer(DatabaseName string) *sqlstore.Container {
	dbLog := waLog.Stdout("Database", "DEBUG", true)

	storeName := fmt.Sprintf("file:%s?_foreign_keys=on", DatabaseName)

	container, err := sqlstore.New("sqlite3", storeName, dbLog)
	if err != nil {
		panic(err)
	}

	return container
}

func NewWhatsAppClientNewDevice(container *sqlstore.Container) *whatsmeow.Client {
	deviceStore := container.NewDevice()
	clientLog := waLog.Stdout("Client", "DEBUG", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)
	return client
}

func GetClientSessionFromContainerADJID(container *sqlstore.Container, operator dbtypes.Operator) (*whatsmeow.Client, error) {
	//TODO TEST
	if operator.OperatorDevice == "" {
		return NewWhatsAppClientNewDevice(container), nil
	}

	//get operator JID from phone
	jid, err := types.ParseJID("51" + operator.OperatorPhone + ":" + operator.OperatorDevice + "@s.whatsapp.net")
	if err != nil {
		panic(err)
	}

	// Search for an existing session
	deviceStore, err := container.GetDevice(jid)
	if err != nil {
		return nil, err
	}

	if deviceStore != nil {
		fmt.Println("JID from device:", *deviceStore.ID)
		clientLog := waLog.Stdout("Client", "DEBUG", true)
		client := whatsmeow.NewClient(deviceStore, clientLog)
		return client, nil
	}

	// If no session is found, create a new one
	return NewWhatsAppClientNewDevice(container), nil
}

func GetClientSessionFromContainerJID(container *sqlstore.Container, operator dbtypes.Operator) (*whatsmeow.Client, error) {

	// Get all devices
	deviceStore, err := container.GetAllDevices()
	if err != nil {
		return nil, err
	}

	// If no devices are found, create a new session
	if len(deviceStore) == 0 {
		return NewWhatsAppClientNewDevice(container), nil
	}

	//add 51 to operator phone
	operatorPhone := "51" + operator.OperatorPhone
	// Search for an existing session
	for _, device := range deviceStore {
		//parse *device.ID into only the number number:device@server, split by : ,get the first element
		DeviceNumber := strings.Split(types.JID.String(*device.ID), ":")[0]
		if operatorPhone == DeviceNumber {
			clientLog := waLog.Stdout("Client", "DEBUG", true)
			client := whatsmeow.NewClient(device, clientLog)
			return client, nil
		}
	}

	// If no session is found, create a new one
	return NewWhatsAppClientNewDevice(container), nil
}

func GetLastChatInteraction(db *sql.DB, conversation dbtypes.Conversation, lead dbtypes.Lead, operator dbtypes.Operator) (string, string) {

	messages, err := nestdb.GetChat(db, conversation.ConversationID)
	if err != nil {
		fmt.Println("Error in get Chat:", err)
		//panic(err)
	}
	var leadMessages, operatorMessages []string

	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]

		if msg.MessageLeadID != uuid.Nil && (msg.MessageLeadID == lead.LeadID) {
			leadMessages = append([]string{msg.MessageText}, leadMessages...)
			lead.LeadID = msg.MessageLeadID
			if len(operatorMessages) > 0 {
				//if the there are messages stacked in the operatorMessages, then the last message is from the operator, and the previous to that is from the lead
				return strings.Join(leadMessages, " "), strings.Join(operatorMessages, " ")
			}
		} else if msg.MessageOperatorID != uuid.Nil && (msg.MessageOperatorID == operator.OperatorID) {
			operatorMessages = append([]string{msg.MessageText}, operatorMessages...)
			operator.OperatorID = msg.MessageOperatorID
			if len(leadMessages) > 0 {
				//if the there are messages stacked in the leadMessages, then the last message is from the lead, and the previous to that is from the operator
				return strings.Join(leadMessages, " "), strings.Join(operatorMessages, " ")

			}

			if lead.LeadID != uuid.Nil && operator.OperatorID != uuid.Nil && msg.MessageLeadID != lead.LeadID && msg.MessageOperatorID != operator.OperatorID {
				break
			}
		}
	}
	return strings.Join(leadMessages, " "), strings.Join(operatorMessages, " ")

}

func LeadHandler(clients map[string]*whatsmeow.Client, whatsContainer *sqlstore.Container, db *sql.DB, lead dbtypes.Lead, operator dbtypes.Operator, conversation dbtypes.Conversation, source string) whatsmeow.EventHandler {
	return func(evt interface{}) {
		switch v := evt.(type) {
		case *events.Message:
			if time.Since(v.Info.Timestamp).Minutes() > 1 { // filter out old messages
				return
			}

			//get lead jid
			leadjid, err := types.ParseJID("51" + lead.LeadPhone + "@s.whatsapp.net")
			if err != nil {
				fmt.Println("Error in parse JID:", err)
				panic(err)
			}

			//lead jid
			sender := v.Info.Sender
			//assert sender and lead are the same
			if sender != leadjid {
				return
			}

			userMessage := v.Message.GetConversation()

			//get last conversation message
			lastmessage, err := nestdb.GetConversationLastMessage(db, conversation.ConversationID)

			//save message to the db
			err = nestdb.AddMessage(db, dbtypes.Message{
				MessageID:           uuid.New(),
				MessageText:         userMessage,
				MessageCreatedAt:    time.Now().Format(time.RFC3339),
				MessageWspTimestamp: v.Info.Timestamp.Format(time.RFC3339),
				MessageParentID:     lastmessage.MessageID,
				MessageLeadID:       lead.LeadID,
				MessageOperatorID:   uuid.Nil, //no id for the operator, it was sent from the lead
				ConversationID:      conversation.ConversationID,
				MessageWspID:        v.Info.ID,
			})

			fmt.Println("Received a message:", userMessage)
			fmt.Println("Sender:", sender)

			lastleadmessage, lastoperatormessage := GetLastChatInteraction(db, conversation, lead, operator)

			//calculate next bot/operator message
			response, finishFlag, brokencycleflag, err := ConversationFlow(lastoperatormessage, lastleadmessage, lead, operator, source)

			client := clients[operator.OperatorPhone]
			//send message only if cycle is not broken
			if !brokencycleflag {

				botMessage := &waProto.Message{Conversation: proto.String(strings.Join([]string{response}, " "))}

				resp, err := client.SendMessage(context.Background(), leadjid, botMessage)
				if err != nil {
					panic(err)
				}
				//print message
				fmt.Println("message sent:", response)

				fmt.Printf("> Message sent: %s\n", resp.ID)

				//save last bot message to the db
				err = nestdb.AddMessage(db, dbtypes.Message{
					MessageID:           uuid.New(),
					MessageText:         response,
					MessageCreatedAt:    time.Now().Format(time.RFC3339),
					MessageWspTimestamp: resp.Timestamp.Format(time.RFC3339),
					MessageParentID:     lastmessage.MessageID,
					MessageLeadID:       uuid.Nil, //no id for the lead, it was sent from the operator
					MessageOperatorID:   operator.OperatorID,
					ConversationID:      conversation.ConversationID,
					MessageWspID:        resp.ID,
				})
			} else if brokencycleflag && response != "" {
				//send message to the lead and to the operator, the cycle is broken, but if response is "" then is because the conversation is over successfully and lead requires further attention

				//lead message
				botMessage := &waProto.Message{Conversation: proto.String(strings.Join([]string{response}, " "))}

				resp, err := client.SendMessage(context.Background(), leadjid, botMessage)
				if err != nil {
					panic(err)
				}
				//print message
				fmt.Println("message sent:", response)

				fmt.Printf("> Message sent: %s\n", resp.ID)

				//save last bot message to the db
				err = nestdb.AddMessage(db, dbtypes.Message{
					MessageID:           uuid.New(),
					MessageText:         response,
					MessageCreatedAt:    time.Now().Format(time.RFC3339),
					MessageWspTimestamp: resp.Timestamp.Format(time.RFC3339),
					MessageParentID:     lastmessage.MessageID,
					MessageLeadID:       uuid.Nil, //no id for the lead, it was sent from the operator
					MessageOperatorID:   operator.OperatorID,
					ConversationID:      conversation.ConversationID,
					MessageWspID:        resp.ID,
				})

				//send message to the operator
				response := fmt.Sprintf("Hola %s,  el cliente %s requiere atención, su nombre es %s", operator.OperatorName, lead.LeadPhone, lead.LeadName)
				fmt.Println("message sent:", response)
				botMessage = &waProto.Message{Conversation: proto.String(strings.Join([]string{response}, " "))}
				operatorpersonaljid, err := types.ParseJID("51" + operator.OperatorPersonalPhone + "@s.whatsapp.net")
				if err != nil {
					fmt.Println("Error in parse JID for operator personal phone:", err)
					panic(err)
				}

				resp, err = client.SendMessage(context.Background(), operatorpersonaljid, botMessage)
				if err != nil {
					panic(err)
				}

			} else {
				//this is where the cycle is broken because the bot can't understand the lead message, brokencycle && response == ""
				//send a message to the private phone of the operator
				response := fmt.Sprintf("Hola %s, el ciclo de conversación se ha roto, por favor, continua la  conversación con el cliente %s, su nombre es %s", operator.OperatorName, lead.LeadPhone, lead.LeadName)
				fmt.Println("message sent:", response)
				botMessage := &waProto.Message{Conversation: proto.String(strings.Join([]string{response}, " "))}
				operatorpersonaljid, err := types.ParseJID("51" + operator.OperatorPersonalPhone + "@s.whatsapp.net")
				if err != nil {
					fmt.Println("Error in parse JID for operator personal phone:", err)
					panic(err)
				}

				resp, err := client.SendMessage(context.Background(), operatorpersonaljid, botMessage)
				if err != nil {
					panic(err)
				}
				fmt.Printf("> Message sent: %s\n", resp.ID)
				//this message is not saved in the database. TODO: save it?

			}

			//TODO notify the backoffice api a new message with the timestamp
			//TODO disconnect client? or keep it connected? what is better for many conversations
			if finishFlag {
				//retreive event handler id from the db
				handlerId, err := nestdb.GetHandler(db, conversation.ConversationID)
				if err != nil {
					fmt.Println("Error in get Handler:", err)
					panic(err)
				}
				go client.RemoveEventHandler(handlerId)
				//erase handler from the db
				err = nestdb.DeleteHandler(db, conversation.ConversationID)
				if err != nil {
					fmt.Println("Error in delete Handler:", err)
					panic(err)
				}
			}
		case *events.Receipt:
			//get lead jid
			leadjid, err := types.ParseJID("51" + lead.LeadPhone + "@s.whatsapp.net")
			if err != nil {
				fmt.Println("Error in parse JID:", err)
				panic(err)
			}

			//get all conversation ID's
			conversationMessages, err := nestdb.GetChat(db, conversation.ConversationID)
			if err != nil {
				fmt.Println("Error in get Chat:", err)
				//panic(err)
			}

			//store only the message IDs
			conversationMessagesIDs := []string{}
			for _, message := range conversationMessages {
				conversationMessagesIDs = append(conversationMessagesIDs, message.MessageWspID)
			}
			var notStoredMessage bool
			for _, messageID := range v.MessageIDs {
				if !contains(conversationMessagesIDs, messageID) {
					notStoredMessage = true
				}
			}
			// //print v
			// fmt.Println("Receipt: ", v)
			// //print sender
			// fmt.Println("Receipt sender: ", v.Sender)
			// //print message source
			// fmt.Println("Receipt message sender ", v.MessageSource.Sender)

			// //print isfromme
			// fmt.Println("Receipt is from me: ", v.MessageSource.IsFromMe)

			// //print message type
			// fmt.Println("Receipt message type: ", v.Type)
			// //print message ID
			// fmt.Println("Receipt message ID: ", v.MessageIDs)
			// //print message IDs

			//if   they're different, and both come from me, then the operator is using a different device, and if the event message is not in the db, then the operation is interrupted
			if !v.MessageSource.IsFromMe && (v.Type == "" || v.Type == "Read") && notStoredMessage && v.Sender == leadjid {

				client := clients[operator.OperatorPhone]

				//retreive event handler id from the db
				handlerId, err := nestdb.GetHandler(db, conversation.ConversationID)
				if err != nil {
					fmt.Println("Error in get Handler:", err)
					panic(err)
				}
				go client.RemoveEventHandler(handlerId)
				//erase handler from the db
				err = nestdb.DeleteHandler(db, conversation.ConversationID)
				if err != nil {
					fmt.Println("Error in delete Handler:", err)
					panic(err)
				} else {
					fmt.Println("Handler deleted, interrupted bot operation")
				}
				fmt.Println("INTERRUPTED OPERATION")
			}
		}
	}
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}
