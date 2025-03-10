package whats

import (
	"fmt"
	"strings"

	"github.com/AlvaroZev/rimont-messager/dbtypes"
)

func ConversationFlow(bot_message string, user_message string, lead dbtypes.Lead, operator dbtypes.Operator, source string) (string, bool, bool, error) {
	if source == "call" {
		return CallConversationFlow(bot_message, user_message, lead, operator)
	} else if source == "feedback" {
		return FeedbackConversationFlow(bot_message, user_message, lead, operator)
	} else if source == "noanswer" {
		return NoAnswerConversationFlow(bot_message, user_message, lead, operator)
	} else {
		return "", false, false, fmt.Errorf("Source not found")
	}
}

func CallConversationFlow(bot_message string, user_message_original string, lead dbtypes.Lead, operator dbtypes.Operator) (string, bool, bool, error) {
	var err error
	var finishConversationMessage bool = false //this true makes the bot to close the conversation
	var notifyOperator bool = false            //this true makes the bot to notify the operator to take a look on this lead
	var processedMessage bool = false          //this true means that the message was processed and we can continue with the flow without looking for more conditions. return?
	operatorMessage := ""
	user_message := strings.ToLower(user_message_original)

	//Addición de regla específica para Incamotors y chevrolet, Se muestra como Novaautos
	if strings.ToLower(lead.LeadInterestBrand) == "chevrolet" && strings.ToLower(lead.LeadDealership) == "incamotors" {
		lead.LeadDealership = "Novaautos"
	}

	bot_message_1 := fmt.Sprintf("¡Hola! %s, nos comunicamos por parte de %s %s %s. Recibimos una solicitud de información por parte de usted, por el modelo %s. Por favor confírmenos si la solicitud es correcta, para poder brindarle toda la información. \n 1. Sí, deseo información \n 2. No, no deseo información", lead.LeadName, lead.LeadInterestBrand, lead.LeadProvince, lead.LeadDealership, lead.LeadInterestModel)
	bot_message_2 := "Excelente, ¿Desea adquirir el vehículo a través de financiamiento o al contado? \n 1. Financiamiento \n 2. Contado"
	bot_message_3 := "Bien ¿Su compra está proyectada para este mes? \n 1. Sí \n 2. No"
	bot_message_4 := "Entiendo ¿Desea que un asesor se comunique con usted mediante una llamada o desea continuar la conversación por WhatsApp? \n 1. Llamada \n 2. WhatsApp"
	bot_message_5 := "Perfecto! Nuestro asesor de ventas se pondrá en contacto con usted mediante la opción elegida."
	operatorMessage = bot_message_1
	// Initial message from the operator
	if bot_message == "" && user_message == "" {
		operatorMessage = bot_message_1
	}

	// Response based on the user message (1st in flow)
	if (AffirmativeMessage(user_message) || strings.Contains(user_message, "1") || strings.Contains(user_message, "uno")) && !processedMessage && bot_message == bot_message_1 {

		operatorMessage = bot_message_2
		processedMessage = true

	} else if (NegativeMessage(user_message) || strings.Contains(user_message, "2") || strings.Contains(user_message, "dos")) && !processedMessage && bot_message == bot_message_1 {

		operatorMessage = "Entendido, muchas gracias."
		processedMessage = true
		finishConversationMessage = true

	}

	// Response based on the user message (2nd in flow)
	if (strings.Contains(user_message, "financiamiento") || strings.Contains(user_message, "1") || strings.Contains(user_message, "uno")) && !processedMessage && bot_message == bot_message_2 {

		operatorMessage = bot_message_3
		processedMessage = true

	} else if (strings.Contains(user_message, "contado") || strings.Contains(user_message, "2") || strings.Contains(user_message, "dos")) && !processedMessage && bot_message == bot_message_2 {

		operatorMessage = bot_message_3
		processedMessage = true

	}

	// Response based on the user message (3rd in flow)
	if (AffirmativeMessage(user_message) || strings.Contains(user_message, "1") || strings.Contains(user_message, "uno")) && !processedMessage && bot_message == bot_message_3 {

		operatorMessage = bot_message_4
		processedMessage = true

	} else if (NegativeMessage(user_message) || strings.Contains(user_message, "2") || strings.Contains(user_message, "dos")) && !processedMessage && bot_message == bot_message_3 {

		operatorMessage = bot_message_4
		processedMessage = true

	}

	// Response based on the user message (4th in flow)
	if (strings.Contains(user_message, "llamada") || strings.Contains(user_message, "1") || strings.Contains(user_message, "uno")) && !processedMessage && bot_message == bot_message_4 {

		operatorMessage = bot_message_5
		//notifyOperator = true uncomment if we need to notify the operator to take a look on this lead
		finishConversationMessage = true
		processedMessage = true
		notifyOperator = true //para notificar al asesor

	} else if (strings.Contains(user_message, "whatsapp") || strings.Contains(user_message, "2") || strings.Contains(user_message, "dos")) && !processedMessage && bot_message == bot_message_4 {

		operatorMessage = bot_message_5
		finishConversationMessage = true
		processedMessage = true
		notifyOperator = true //para notificar al asesor

	}

	if !processedMessage && user_message != "" { //if message has not been processed, could not fit into the flow and is not empty
		operatorMessage = ""
		notifyOperator = true //para notificar al asesor
		finishConversationMessage = true
		processedMessage = true
	}

	//TODO?
	return operatorMessage, finishConversationMessage, notifyOperator, err

}

func NoAnswerConversationFlow(botMessage string, user_message_original string, lead dbtypes.Lead, operator dbtypes.Operator) (string, bool, bool, error) {
	var err error
	var finishConversationMessage bool
	var notifyOperator bool = false
	var processedMessage bool
	user_message := strings.ToLower(user_message_original)

	// Initial greeting and first question
	bot_message_1 := fmt.Sprintf("¡Hola! %s, le escribimos por parte de %s %s %s, queríamos brindarle la información solicitada sobre el modelo %s. ¿Aun se encuentra interesado en adquirir la unidad?\n1. Sí, aún estoy interesado y me gustaría recibir información.\n2. No, ya no tengo interés.\n3. Si estoy interesado, pero en otro modelo.", lead.LeadName, lead.LeadInterestBrand, lead.LeadProvince, lead.LeadDealership, lead.LeadInterestModel)

	// Define other messages based on the conversation flow
	messageInterested := "¿Desea que un asesor se comunique con usted mediante llamada o por desea continuar por WhatsApp?\n1. Llamada.\n2. WhatsApp."
	messageInterested2 := "Entendido, un asesor de venta se contactará con usted para brindarle la información y nuestras promociones activas, gracias."
	messageNotInterested := "Entiendo, Muchas gracias."
	messageAnotherModel := "Bien ¿Cuál sería el modelo de interés?"
	messageAnotherModel2 := "Entendido, un asesor de venta se contactará con usted para brindarle la información y nuestras promociones activas, gracias."

	operatorMessage := ""

	// Logic to handle the conversation flow based on user responses
	if botMessage == "" && user_message == "" {
		operatorMessage = bot_message_1
		processedMessage = true
	} else if !processedMessage && (AffirmativeMessage(user_message) || strings.Contains(user_message, "1") || strings.Contains(user_message, "uno") || strings.Contains(user_message, "sí, aún estoy interesado")) && botMessage == bot_message_1 {
		operatorMessage = messageInterested
		processedMessage = true
	} else if !processedMessage && (NegativeMessage(user_message) || strings.Contains(user_message, "2") || strings.Contains(user_message, "dos") || strings.Contains(user_message, "no, ya no tengo interés")) && botMessage == bot_message_1 {
		operatorMessage = messageNotInterested
		finishConversationMessage = true
		processedMessage = true
	} else if !processedMessage && (strings.Contains(user_message, "3") || strings.Contains(user_message, "tres") || strings.Contains(user_message, "si estoy interesado, pero en otro modelo") || strings.Contains(user_message, "otro")) && botMessage == bot_message_1 {
		operatorMessage = messageAnotherModel
		processedMessage = true
	} else if !processedMessage && botMessage == messageInterested {
		operatorMessage = messageInterested2
		finishConversationMessage = true
		processedMessage = true
		notifyOperator = true //para notificar al asesor
	} else if !processedMessage && botMessage == messageAnotherModel {
		operatorMessage = messageAnotherModel2
		finishConversationMessage = true
		processedMessage = true
		notifyOperator = true //para notificar al asesor
	}

	// Check if the conversation cannot continue based on the user's response
	if !processedMessage && user_message != "" {
		operatorMessage = ""
		notifyOperator = true
		finishConversationMessage = true
		processedMessage = true
	}

	return operatorMessage, finishConversationMessage, notifyOperator, err
}

func FeedbackConversationFlow(botMessage string, user_message_original string, lead dbtypes.Lead, operator dbtypes.Operator) (string, bool, bool, error) {
	var err error
	var finishConversationMessage bool
	var notifyOperator bool = false
	var processedMessage bool
	user_message := strings.ToLower(user_message_original)

	// Initial greeting and first question
	bot_message_1 := fmt.Sprintf("¡Hola! %s, le saluda %s nos comunicamos por parte de %s %s %s queríamos preguntarle ¿Asistió a tienda para ver el modelo %s?\n1. Sí, me acerque a tienda.\n2. No, no me acerque a tienda.", lead.LeadName, operator.OperatorName, lead.LeadInterestBrand, lead.LeadProvince, lead.LeadDealership, lead.LeadInterestModel)

	// Define other messages based on the conversation flow
	messageVisitedStore := "¿Fue atendido de forma satisfactoria en tienda?\n1. Sí, me atendieron bien.\n2. No, no me atendieron bien."
	messageSatisfactoryService := "Excelente ¿Pudo concretar la venta del vehículo?\n1. Sí.\n2. No."
	messageUnsatisfactoryService := "Lamento lo sucedido ¿Cuál fue el motivo por el que no fue atendido bien?\n1. Mala atención del vendedor.\n2. No me brindaron la información correcta.\n3. No había stock de la unidad."
	messageNotVisitedStore := "¿Cuál fue el motivo por el que no se pudo acercar a tienda?\n1. No tuve tiempo.\n2. Perdida de Interés.\n3. Otros."
	messageNotVisitedStoreFollowUp := "Entiendo ¿Desea reagendar una cita en el concesionario?\n1. Sí, por favor.\n2. No, gracias."
	messageRescheduleConfirmation := "Entendido"
	messageNoReschedule := "Entiendo, Muchas gracias."
	messageProblemReported := "Entiendo, se reportará lo sucedido."

	operatorMessage := ""

	// Logic to handle the conversation flow based on user responses
	if botMessage == "" && user_message == "" {
		operatorMessage = bot_message_1
		processedMessage = true
	} else if !processedMessage && (AffirmativeMessage(user_message) || strings.Contains(user_message, "1") || strings.Contains(user_message, "uno") || strings.Contains(user_message, "sí, me acerque a tienda")) && botMessage == bot_message_1 {
		operatorMessage = messageVisitedStore
		processedMessage = true
	} else if !processedMessage && (NegativeMessage(user_message) || strings.Contains(user_message, "2") || strings.Contains(user_message, "dos") || strings.Contains(user_message, "no, no me acerque a tienda")) && botMessage == bot_message_1 {
		operatorMessage = messageNotVisitedStore
		processedMessage = true
	} else if !processedMessage && (AffirmativeMessage(user_message) || strings.Contains(user_message, "1") || strings.Contains(user_message, "uno") || strings.Contains(user_message, "sí, me atendieron bien")) && botMessage == messageVisitedStore {
		operatorMessage = messageSatisfactoryService
		processedMessage = true
	} else if !processedMessage && (NegativeMessage(user_message) || strings.Contains(user_message, "2") || strings.Contains(user_message, "dos") || strings.Contains(user_message, "no, no me atendieron bien")) && botMessage == messageVisitedStore {
		operatorMessage = messageUnsatisfactoryService
		processedMessage = true
	} else if !processedMessage && botMessage == messageUnsatisfactoryService {
		operatorMessage = messageProblemReported
		finishConversationMessage = true
		processedMessage = true
	} else if !processedMessage && botMessage == messageSatisfactoryService {
		operatorMessage = "Entiendo"
		finishConversationMessage = true
		processedMessage = true
	} else if !processedMessage && (strings.Contains(user_message, "2") || strings.Contains(user_message, "dos") || strings.Contains(user_message, "perdida de interés")) && botMessage == messageNotVisitedStore { //P1 and answers Rsp 2.1.2
		operatorMessage = "Entiendo, Muchas gracias." //answer with RSP 2.1.2.1 and close
		processedMessage = true
		finishConversationMessage = true
	} else if !processedMessage && botMessage == messageNotVisitedStore { //P1
		operatorMessage = messageNotVisitedStoreFollowUp //P2
		processedMessage = true
	} else if !processedMessage && (AffirmativeMessage(user_message) || strings.Contains(user_message, "1") || strings.Contains(user_message, "uno") || strings.Contains(user_message, "sí, por favor")) && botMessage == messageNotVisitedStoreFollowUp {
		operatorMessage = messageRescheduleConfirmation
		finishConversationMessage = true
		processedMessage = true
		notifyOperator = true //para notificar al asesor
	} else if !processedMessage && (NegativeMessage(user_message) || strings.Contains(user_message, "2") || strings.Contains(user_message, "dos") || strings.Contains(user_message, "no, gracias")) && botMessage == messageNotVisitedStoreFollowUp {
		operatorMessage = messageNoReschedule
		finishConversationMessage = true
		processedMessage = true
	}

	// Check if the conversation cannot continue based on the user's response
	if !processedMessage && user_message != "" {
		operatorMessage = ""
		notifyOperator = true
		finishConversationMessage = true
		processedMessage = true
	}

	return operatorMessage, finishConversationMessage, notifyOperator, err
}

func AffirmativeMessage(s string) bool {
	// List of common affirmative words/phrases in Spanish
	affirmativeWords := []string{"sí", "si", "claro", "por supuesto", "seguro", "exacto", "correcto", "desde luego", "ok", "okay", "ya", "yap", "perfecto", "Listo"}

	// Normalize the input string (to lowercase for comparison)
	normalizedString := strings.ToLower(s)

	// Check if any affirmative word is in the string
	for _, word := range affirmativeWords {
		if strings.Contains(normalizedString, word) {
			return true
		}
	}

	return false
}

func NegativeMessage(s string) bool {
	// List of common negative words/phrases in Spanish
	negativeWords := []string{
		"no", "nunca", "jamás", "nada", "nadie", "ninguno", "ninguna", "ni", "tampoco",
		"imposible", "difícil", "problema", "mal", "peor", "nunca más", "sin", "jamás de los jamases",
		"en absoluto", "de ninguna manera", "de ningún modo", "ni en sueños", "ni hablar",
		"ni pensarlo", "rechazo", "negativo", "ningún caso", "ninguna vez",
	}

	// Normalize the input string (to lowercase for comparison)
	normalizedString := strings.ToLower(s)

	// Check if any negative word is in the string
	for _, word := range negativeWords {
		if strings.Contains(normalizedString, word) {
			return true
		}
	}

	return false
}
