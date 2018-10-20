package model

import (
	"errors"
	"github.com/openaccounting/oa-server/core/model/types"
	"github.com/openaccounting/oa-server/core/util"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"log"
)

type UserInterface interface {
	CreateUser(user *types.User) error
	VerifyUser(string) error
	UpdateUser(user *types.User) error
	ResetPassword(email string) error
	ConfirmResetPassword(string, string) (*types.User, error)
}

func (model *Model) CreateUser(user *types.User) error {
	if user.Id == "" {
		return errors.New("id required")
	}

	if user.FirstName == "" {
		return errors.New("first name required")
	}

	if user.LastName == "" {
		return errors.New("last name required")
	}

	if user.Email == "" {
		return errors.New("email required")
	}

	if user.Password == "" {
		return errors.New("password required")
	}

	if user.AgreeToTerms != true {
		return errors.New("must agree to terms")
	}

	// hash password
	// bcrypt's function also generates a salt

	passwordHash, err := model.bcrypt.GenerateFromPassword([]byte(user.Password), model.bcrypt.GetDefaultCost())
	if err != nil {
		return err
	}

	user.PasswordHash = string(passwordHash)
	user.Password = ""
	user.EmailVerified = false
	user.EmailVerifyCode, err = util.NewGuid()

	if err != nil {
		return err
	}

	err = model.db.InsertUser(user)

	if err != nil {
		return err
	}

	err = model.SendVerificationEmail(user)

	if err != nil {
		log.Println(err)
	}

	return nil
}

func (model *Model) VerifyUser(code string) error {
	if code == "" {
		return errors.New("code required")
	}

	return model.db.VerifyUser(code)
}

func (model *Model) UpdateUser(user *types.User) error {
	if user.Id == "" {
		return errors.New("id required")
	}

	if user.Password == "" {
		return errors.New("password required")
	}

	// hash password
	// bcrypt's function also generates a salt

	passwordHash, err := model.bcrypt.GenerateFromPassword([]byte(user.Password), model.bcrypt.GetDefaultCost())
	if err != nil {
		return err
	}

	user.PasswordHash = string(passwordHash)
	user.Password = ""

	return model.db.UpdateUser(user)
}

func (model *Model) ResetPassword(email string) error {
	if email == "" {
		return errors.New("email required")
	}

	user, err := model.db.GetVerifiedUserByEmail(email)

	if err != nil {
		// Don't send back error so people can't try to find user accounts
		log.Printf("Invalid email for reset password " + email)
		return nil
	}

	user.PasswordReset, err = util.NewGuid()

	if err != nil {
		return err
	}

	err = model.db.UpdateUserResetPassword(user)

	if err != nil {
		return err
	}

	return model.SendPasswordResetEmail(user)
}

func (model *Model) ConfirmResetPassword(password string, code string) (*types.User, error) {
	if password == "" {
		return nil, errors.New("password required")
	}

	if code == "" {
		return nil, errors.New("code required")
	}

	user, err := model.db.GetUserByResetCode(code)

	if err != nil {
		return nil, errors.New("Invalid code")
	}

	passwordHash, err := model.bcrypt.GenerateFromPassword([]byte(password), model.bcrypt.GetDefaultCost())
	if err != nil {
		return nil, err
	}

	user.PasswordHash = string(passwordHash)
	user.Password = ""

	err = model.db.UpdateUser(user)

	if err != nil {
		return nil, err
	}

	return user, nil
}

func (model *Model) SendVerificationEmail(user *types.User) error {
	log.Println("Sending verification email to " + user.Email)

	link := model.config.WebUrl + "/user/verify?code=" + user.EmailVerifyCode

	from := mail.NewEmail(model.config.SendgridSender, model.config.SendgridEmail)
	subject := "Verify your email"
	to := mail.NewEmail(user.FirstName+" "+user.LastName, user.Email)

	plainTextContent := "Thank you for signing up with Open Accounting! " +
		"Please click on the link below to verify your email address:\n\n" + link
	htmlContent := "Thank you for signing up with Open Accounting! " +
		"Please click on the link below to verify your email address:<br><br>" +
		"<a href=\"" + link + "\">" + link + "</a>"

	message := mail.NewSingleEmail(from, subject, to, plainTextContent, htmlContent)
	client := sendgrid.NewSendClient(model.config.SendgridKey)
	response, err := client.Send(message)

	if err != nil {
		return err
	}

	log.Println(response.StatusCode)
	log.Println(response.Body)
	log.Println(response.Headers)

	return nil
}

func (model *Model) SendPasswordResetEmail(user *types.User) error {
	log.Println("Sending password reset email to " + user.Email)

	link := model.config.WebUrl + "/user/reset-password?code=" + user.PasswordReset

	from := mail.NewEmail(model.config.SendgridSender, model.config.SendgridEmail)
	subject := "Reset password"
	to := mail.NewEmail(user.FirstName+" "+user.LastName, user.Email)

	plainTextContent := "Please click the following link to reset your password:\n\n" + link +
		"If you did not request to have your password reset, please ignore this email and " +
		"nothing will happen."
	htmlContent := "Please click the following link to reset your password:<br><br>\n" +
		"<a href=\"" + link + "\">" + link + "</a><br><br>\n" +
		"If you did not request to have your password reset, please ignore this email and " +
		"nothing will happen."

	message := mail.NewSingleEmail(from, subject, to, plainTextContent, htmlContent)
	client := sendgrid.NewSendClient(model.config.SendgridKey)
	response, err := client.Send(message)

	if err != nil {
		return err
	}

	log.Println(response.StatusCode)
	log.Println(response.Body)
	log.Println(response.Headers)

	return nil
}