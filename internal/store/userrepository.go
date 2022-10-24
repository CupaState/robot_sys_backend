package store

import (
	"database/sql"

	pb "gowallet/cmd/psg_worker/proto"
	"gowallet/internal/model"
)

type UserRepository struct {
	store *Store
}

func (r *UserRepository) Create(u *pb.UserModel) (error) {
	if err := model.Validate(u); err != nil {
		return err
	}

	u, err := model.BeforeCreate(u)
	if err != nil {
		return err
	}

	return r.store.db.QueryRow(
		"INSERT INTO users (username, encrypted_password, email, external_wallet_addr, 	created_on) VALUES ($1, $2, $3, $4, $5) RETURNING user_id",
		u.UserName,
		u.EncryptedPassword,
		u.Email,
		u.ExternalWalletAddr,
		u.CreatedOn.AsTime(),
	).Scan(&u.UserId)
}

func (r *UserRepository) FindByEmail(email string) (*pb.UserModel, error) {
	u := &pb.UserModel{}
	err := r.store.db.QueryRow(
		"SELECT * FROM users WHERE email = $1", 
		email).Scan(
			&u.UserId,
			&u.UserName,
			&u.EncryptedPassword,
			&u.Email,
			&u.ExternalWalletAddr,
			&u.Gain,
			&u.TotalGain,
			&u.PaidMoney,
			&u.CreatedOn,
			&u.LastLogin,
		)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrRecordNotFound
		}
		
		return nil, err
	}

	return u, nil
}
