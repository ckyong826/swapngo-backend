package bizs

import (
	"fmt"

	"swapngo-backend/internal/models"
	"swapngo-backend/pkg/utils"
)

// verifyPin checks a 4-digit transaction PIN against the user's stored bcrypt
// hash. Used by every money-moving biz (deposit/withdraw/swap/transfer) before
// it emits an event.
//
// ponytail: no attempt limit — 10,000 combos are brute-forceable by anyone
// holding a valid JWT. Add a pin_attempts counter + lockout on the user row
// before this goes anywhere near real money.
func verifyPin(user *models.User, pin string) error {
	ok, err := utils.CheckPassword(user.PinHash, pin)
	if err != nil || !ok {
		return fmt.Errorf("invalid PIN")
	}
	return nil
}
