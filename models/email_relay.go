package models

import (
	"time"

	"github.com/google/uuid"
)

// EmailRelay records one inbound message from an external address to one of
// our @<domain> aliases, so a later reply from the personal inbox can be
// routed back out to the right person, from the right alias, without ever
// exposing the personal address. See controllers/mail.go.
type EmailRelay struct {
	Id              uuid.UUID `db:"id"`
	AliasAddress    string    `db:"alias_address"`
	ExternalAddress string    `db:"external_address"`
	ExternalName    string    `db:"external_name"`
	Subject         string    `db:"subject"`
	MessageId       string    `db:"message_id"`
	CreatedAt       time.Time `db:"created_at"`
	LastUsedAt      time.Time `db:"last_used_at"`
}
