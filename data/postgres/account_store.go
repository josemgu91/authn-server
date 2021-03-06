package postgres

import (
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/keratin/authn-server/models"
)

type AccountStore struct {
	*sqlx.DB
}

func (db *AccountStore) Find(id int) (*models.Account, error) {
	account := models.Account{}
	err := db.Get(&account, "SELECT * FROM accounts WHERE id = $1", id)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	if account.DeletedAt != nil {
		account.Username = ""
	}
	return &account, nil
}

func (db *AccountStore) FindByUsername(u string) (*models.Account, error) {
	account := models.Account{}
	err := db.Get(&account, "SELECT * FROM accounts WHERE username = $1 AND deleted_at IS NULL", u)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &account, nil
}

func (db *AccountStore) FindByOauthAccount(provider string, providerID string) (*models.Account, error) {
	account := models.Account{}
	err := db.Get(&account, "SELECT a.* FROM accounts a INNER JOIN oauth_accounts oa ON a.id = oa.account_id WHERE oa.provider = $1 AND oa.provider_id = $2", provider, providerID)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &account, nil
}

func (db *AccountStore) Create(u string, p []byte) (*models.Account, error) {
	now := time.Now()

	account := &models.Account{
		Username:          u,
		Password:          p,
		PasswordChangedAt: now,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	result, err := db.NamedQuery(
		`INSERT INTO accounts (
			username, 
			password, 
			locked, 
			require_new_password, 
			password_changed_at, 
			created_at, 
			updated_at
		) 
		VALUES (:username, :password, :locked, :require_new_password, :password_changed_at, :created_at, :updated_at)
		RETURNING id`,
		account,
	)
	if err != nil {
		return nil, err
	}
	result.Next()
	var id int64
	err = result.Scan(&id)
	if err != nil {
		return nil, err
	}
	account.ID = int(id)

	return account, nil
}

func (db *AccountStore) AddOauthAccount(accountID int, provider string, providerID string, accessToken string) error {
	now := time.Now()

	_, err := db.NamedExec(`
        INSERT INTO oauth_accounts (account_id, provider, provider_id, access_token, created_at, updated_at)
        VALUES (:account_id, :provider, :provider_id, :access_token, :created_at, :updated_at)
    `, map[string]interface{}{
		"account_id":   accountID,
		"provider":     provider,
		"provider_id":  providerID,
		"access_token": accessToken,
		"created_at":   now,
		"updated_at":   now,
	})
	return err
}

func (db *AccountStore) GetOauthAccounts(accountID int) ([]*models.OauthAccount, error) {
	accounts := []*models.OauthAccount{}
	err := db.Select(&accounts, `SELECT * FROM oauth_accounts WHERE account_id = $1`, accountID)
	return accounts, err
}

func (db *AccountStore) Archive(id int) error {
	_, err := db.Exec("DELETE FROM oauth_accounts WHERE account_id = $1", id)
	if err != nil {
		return err
	}
	_, err = db.Exec(`
		UPDATE accounts
		SET
			username = CONCAT('@', MD5(RANDOM()::TEXT)),
			password = $1,
			deleted_at = $2
		WHERE id = $3`, "", time.Now(), id)
	return err
}

func (db *AccountStore) Lock(id int) error {
	_, err := db.Exec("UPDATE accounts SET locked = $1, updated_at = $2 WHERE id = $3", true, time.Now(), id)
	return err
}

func (db *AccountStore) Unlock(id int) error {
	_, err := db.Exec("UPDATE accounts SET locked = $1, updated_at = $2 WHERE id = $3", false, time.Now(), id)
	return err
}

func (db *AccountStore) RequireNewPassword(id int) error {
	_, err := db.Exec("UPDATE accounts SET require_new_password = $1, updated_at = $2 WHERE id = $3", true, time.Now(), id)
	return err
}

func (db *AccountStore) SetPassword(id int, p []byte) error {
	_, err := db.Exec(`
		UPDATE accounts 
		SET
			password = $1,
			require_new_password = $2,
			password_changed_at = $3, 
			updated_at = $4
		WHERE 
			id = $5`, p, false, time.Now(), time.Now(), id)
	return err
}

func (db *AccountStore) UpdateUsername(id int, u string) error {
	_, err := db.Exec("UPDATE accounts SET username = $1, updated_at = $2 WHERE id = $3", u, time.Now(), id)
	return err
}
