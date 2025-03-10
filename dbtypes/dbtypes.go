package dbtypes

import (
	"github.com/google/uuid"
)

type Lead struct {
	LeadID            uuid.UUID `json:"lead_id"`
	LeadName          string    `json:"lead_name"`
	LeadSurnames      string    `json:"lead_surnames"`
	LeadPhone         string    `json:"lead_phone"`
	LeadInterestBrand string    `json:"lead_interest_brand"`
	LeadInterestModel string    `json:"lead_interest_model"`
	LeadProvince      string    `json:"lead_province"`
	LeadDealership    string    `json:"lead_dealership"`
	LeadBackOfficeID  string    `json:"lead_back_office_id"`
}

type Operator struct {
	OperatorID            uuid.UUID `json:"operator_id"`
	OperatorName          string    `json:"operator_name"`
	OperatorSurnames      string    `json:"operator_surnames"`
	OperatorPhone         string    `json:"operator_phone"`
	OperatorBackOfficeID  string    `json:"operator_back_office_id"`
	OperatorDevice        string    `json:"operator_device"`
	OperatorPersonalPhone string    `json:"operator_personal_phone"`
}

type Message struct {
	MessageID           uuid.UUID `json:"message_id"`
	MessageText         string    `json:"message_text"`
	MessageCreatedAt    string    `json:"message_created_at"`
	MessageWspTimestamp string    `json:"message_wsp_timestamp"`
	MessageParentID     uuid.UUID `json:"message_parent_id"`
	MessageLeadID       uuid.UUID `json:"message_lead_id"`
	MessageOperatorID   uuid.UUID `json:"message_operator_id"`
	ConversationID      uuid.UUID `json:"conversation_id"`
	MessageWspID        string    `json:"message_wsp_id"`
}

type Conversation struct {
	ConversationID uuid.UUID `json:"conversation_id"`
	LeadID         uuid.UUID `json:"lead_id"`
	OperatorID     uuid.UUID `json:"operator_id"`
}
