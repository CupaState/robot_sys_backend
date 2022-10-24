package model

// METHODS TO WORK WITH USER MODEL

import (
	pb "gowallet/cmd/psg_worker/proto"

	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/go-ozzo/ozzo-validation/is"
	"golang.org/x/crypto/bcrypt"
)

func Validate(u *pb.UserModel) error {
	return validation.ValidateStruct(
		u,
		validation.Field(&u.Email, validation.Required, is.Email, validation.Length(6, 100)),
		validation.Field(&u.Password, validation.By(requiredIf(u.EncryptedPassword == "")), validation.Length(6, 100)),
	)
}

// ClearPassword ...
func ClearPassword(u *pb.UserModel) *pb.UserModel {
	u.Password = ""
	return u
}

// ComparePassword ...
func ComparePassword(u *pb.UserModel, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(u.EncryptedPassword), []byte(password)) == nil
}
