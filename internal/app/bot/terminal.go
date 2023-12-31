package bot

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"

	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
	"golang.org/x/term"
)

type Terminal struct {
	PhoneNumber string
}

func (Terminal) SignUp(ctx context.Context) (auth.UserInfo, error) {
	return auth.UserInfo{}, errors.New("signing up not implemented in Terminal")
}

func (Terminal) AcceptTermsOfService(ctx context.Context, tos tg.HelpTermsOfService) error {
	return &auth.SignUpRequired{TermsOfService: tos}
}

func (t Terminal) Code(ctx context.Context, sentCode *tg.AuthSentCode) (string, error) {
	fmt.Printf("Enter %s code: ", t.PhoneNumber)
	code, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(code), nil
}

func (t Terminal) Phone(_ context.Context) (string, error) {
	return t.PhoneNumber, nil
}

func (t Terminal) Password(_ context.Context) (string, error) {
	fmt.Printf("Enter %s 2FA password: ", t.PhoneNumber)
	bytePwd, err := term.ReadPassword(syscall.Stdin)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(bytePwd)), nil
}

type ProcTerminal struct {
	PhoneNumber string

	FCode     chan string
	FPassword chan string
}

func (ProcTerminal) SignUp(ctx context.Context) (auth.UserInfo, error) {
	return auth.UserInfo{}, errors.New("signing up not implemented in Terminal")
}

func (ProcTerminal) AcceptTermsOfService(ctx context.Context, tos tg.HelpTermsOfService) error {
	return &auth.SignUpRequired{TermsOfService: tos}
}

func (t ProcTerminal) Code(ctx context.Context, sentCode *tg.AuthSentCode) (string, error) {
	code := <-t.FCode
	log.Println("got code")
	return strings.TrimSpace(code), nil
}

func (t ProcTerminal) Phone(_ context.Context) (string, error) {
	return t.PhoneNumber, nil
}

func (t ProcTerminal) Password(_ context.Context) (string, error) {
	password := <-t.FPassword
	return strings.TrimSpace(password), nil
}
