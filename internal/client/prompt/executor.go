package prompt

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smanhack/gophkeeper/internal/client/app"
	"github.com/smanhack/gophkeeper/internal/client/model"
)

type Executor struct {
	app        *app.App
	currentCtx context.Context
	mu         sync.RWMutex
}

func (e *Executor) getContext() context.Context {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.currentCtx
}

func (e *Executor) updateContext(ctx context.Context) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.currentCtx = ctx
}

// NewExecutor creates executor and returns Execute function
func NewExecutor(ctx context.Context, cancel context.CancelFunc) func(string) {
	appL, appCancel, err := app.NewApp(ctx)
	if err != nil {
		panic(err)
	}

	e := &Executor{
		app:        appL,
		currentCtx: ctx,
	}

	return func(s string) {
		e.execute(s, appCancel, cancel)
	}
}

func (e *Executor) execute(s string, appCancel, mainCancel context.CancelFunc) {
	var isForce bool

	setCommand, options := getCommandArgsAndOptions(s)
	if options["force"] || options["f"] {
		isForce = true
	}

	ctx := e.getContext()

	switch setCommand[0] {
	case "login":
		if err := e.login(ctx, setCommand); err != nil {
			fmt.Println(err)
			return
		}

		fmt.Println("successfully authorized")
		return
	case "register":
		if err := e.register(ctx, setCommand); err != nil {
			fmt.Println(err)
			return
		}

		fmt.Println("User successfully created. You are logged in.")

		return
	case "delete-user":
		if err := e.deleteUser(ctx); err != nil {
			fmt.Println(err)
			return
		}

		fmt.Println("you successfully deleted account and logged out")

		return
	case "logout":
		if err := e.logout(ctx); err != nil {
			fmt.Println(err)
			return
		}

		fmt.Println("you successfully logged out")
		return
	case "types":
		types, err := e.types(ctx)
		if err != nil {
			fmt.Println(err)
			return
		}

		for _, t := range types {
			fmt.Printf("%+v\n", t)
		}

		return
	case "create-auth":
		if err := e.createAuth(ctx, setCommand); err != nil {
			fmt.Println(err)
			return
		}
		return
	case "create-text":
		if err := e.createText(ctx, setCommand); err != nil {
			fmt.Println(err)
			return
		}

		return
	case "create-binary":
		if err := e.createBinary(ctx, setCommand); err != nil {
			fmt.Println(err)
			return
		}

		return
	case "create-card":
		if err := e.createCard(ctx, setCommand); err != nil {
			fmt.Println(err)
			return
		}

		return
	case "delete-secret":
		if err := e.deleteSecret(ctx, setCommand); err != nil {
			fmt.Println(err)
			return
		}

		return
	case "get-secrets-by-type":
		list, err := e.getSecretsByTypeId(ctx, setCommand)
		if err != nil {
			fmt.Println(err)
			return
		}

		for _, secret := range list {
			fmt.Printf("ID:%v Title: %v\n", secret.Id, secret.Name)
		}

		return
	case "get-secret":
		secret, err := e.getSecret(ctx, setCommand)
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Printf("Content:%+v\n", secret)

		return
	case "get-secret-binary":
		if err := e.getSecretBinary(ctx, setCommand); err != nil {
			fmt.Println(err)
			return
		}
		return
	case "edit-secret":
		if err := e.editSecret(ctx, setCommand, isForce); err != nil {
			fmt.Println(err)
			return
		}

		return
	case "exit":
		appCancel()
		mainCancel()
		e.app.Cron.Stop()

		os.Exit(0)
	}
}

func (e *Executor) login(ctx context.Context, args []string) error {
	switch len(args) - 1 {
	case 1:
		return fmt.Errorf("validation error: Password is missing")
	case 0:
		return fmt.Errorf("validation error: Login and Password is missing")
	}

	account := model.Account{Username: args[1], Credential: args[2]}

	newCtx, err := e.app.AccountService.Authenticate(ctx, account)
	if err != nil {
		st, _ := status.FromError(err)

		switch st.Code() {
		case codes.NotFound:
			return fmt.Errorf("error: User not found")
		default:
			return fmt.Errorf("error:" + st.Message())
		}
	}

	e.updateContext(newCtx)

	e.app.Syncer.SyncAll(newCtx)

	go e.app.Cron.Run()

	return nil
}

func (e *Executor) register(ctx context.Context, args []string) error {
	switch len(args) - 1 {
	case 1:
		return fmt.Errorf("validation error: Password is missing")
	case 0:
		return fmt.Errorf("validation error: Login and Password is missing")
	}

	account := model.Account{Username: args[1], Credential: args[2]}
	newCtx, err := e.app.AccountService.SignUp(ctx, account)
	if err != nil {
		switch status.Code(err) {
		case codes.InvalidArgument:
			return fmt.Errorf("error: data is invalid or user already exists")
		default:
			return err
		}
	}

	e.updateContext(newCtx)

	return nil
}

func (e *Executor) deleteUser(ctx context.Context) error {
	newCtx, err := e.app.AccountService.Remove(ctx)
	if err != nil {
		return err
	}
	e.updateContext(newCtx)
	return nil
}

func (e *Executor) logout(ctx context.Context) error {
	newCtx := e.app.AccountService.Logout(ctx)
	e.updateContext(newCtx)

	e.app.Cron.Stop()

	e.app.Storage.ResetStorage()

	return nil
}

func (e *Executor) types(ctx context.Context) ([]model.DataCategory, error) {
	categories, err := e.app.CategoryService.List(ctx)
	if err != nil {
		return nil, err
	}

	var models []model.DataCategory
	for _, category := range categories.Categories {
		models = append(models, model.DataCategory{
			Id:   int(category.Id),
			Name: category.Title,
		})
	}

	return models, nil
}

func (e *Executor) createAuth(ctx context.Context, args []string) error {
	switch len(args) - 1 {
	case 2:
		return fmt.Errorf("validation error: Password is missing")
	case 1:
		return fmt.Errorf("validation error: Login and Password is missing")
	case 0:
		return fmt.Errorf("validation error: Title, Login, Password is missing")
	}

	m := model.LoginPassSecret{
		Name:       args[1],
		RecordType: 1,
		Login:      args[2],
		Password:   args[3],
	}

	cont, errMarshal := json.Marshal(m)
	if errMarshal != nil {
		return errMarshal
	}

	if err := e.app.DataVaultService.StoreData(ctx, m.Name, 1, string(cont)); err != nil {
		return err
	}

	return nil
}

func (e *Executor) createText(ctx context.Context, args []string) error {
	switch len(args) - 1 {
	case 1:
		return fmt.Errorf("validation error: Text is missing")
	case 0:
		return fmt.Errorf("validation error: Title and Text is missing")
	}

	m := model.TextSecret{
		Name:       args[1],
		RecordType: 2,
		Text:       strings.Join(args[2:], " "),
	}

	marshal, errMarshal := json.Marshal(m)
	if errMarshal != nil {
		return errMarshal
	}

	if err := e.app.DataVaultService.StoreData(ctx, m.Name, 2, string(marshal)); err != nil {
		return err
	}

	return nil
}

func (e *Executor) createBinary(ctx context.Context, args []string) error {
	switch len(args) - 1 {
	case 1:
		return fmt.Errorf("validation error: Filepath is missing")
	case 0:
		return fmt.Errorf("validation error: Title and Filepath is missing")
	}

	m := model.FileSecret{
		Name:       args[1],
		RecordType: 3,
		Path:       args[2],
	}

	f, err := os.Open(m.Path)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			fmt.Printf("Warning: failed to close file: %v\n", closeErr)
		}
	}()

	data, errData := os.ReadFile(m.Path)
	if errData != nil {
		return errData
	}

	errCreate := e.app.DataVaultService.StoreData(ctx, m.Name, m.RecordType, string(data))
	if errCreate != nil {
		return errCreate
	}

	return nil
}

func (e *Executor) createCard(ctx context.Context, args []string) error {
	switch len(args) - 1 {
	case 3:
		return fmt.Errorf("validation error: Due date is missing")
	case 2:
		return fmt.Errorf("validation error: CVV and Due date is missing")
	case 1:
		return fmt.Errorf("validation error: Card number, CVV, Due date is missing")
	case 0:
		return fmt.Errorf("validation error: Title, Card number, CVV, Due date is missing")
	}

	cardModel := model.CardSecret{
		Name:       args[1],
		RecordType: 4,
		CardNumber: args[2],
		CVV:        args[3],
		Due:        args[4],
	}

	cont, er := json.Marshal(cardModel)
	if er != nil {
		return er
	}

	if err := e.app.DataVaultService.StoreData(ctx, cardModel.Name, cardModel.RecordType, string(cont)); err != nil {
		return err
	}

	return nil
}

func (e *Executor) deleteSecret(ctx context.Context, args []string) error {
	switch len(args) - 1 {
	case 0:
		return fmt.Errorf("validation error: Secret ID is missing")
	}

	id, convErr := strconv.Atoi(args[1])
	if convErr != nil {
		return convErr
	}

	if err := e.app.DataVaultService.RemoveData(ctx, id); err != nil {
		return err
	}

	return nil
}

func (e *Executor) getSecretsByTypeId(ctx context.Context, args []string) ([]model.SecretList, error) {
	switch len(args) - 1 {
	case 0:
		return nil, fmt.Errorf("validation error: Secret Type ID is missing")
	}

	id, convErr := strconv.Atoi(args[1])
	if convErr != nil {
		return nil, convErr
	}

	list, err := e.app.DataVaultService.GetListOfDataRecords(ctx, id)
	if err != nil {
		return nil, err
	}

	var models []model.SecretList
	for _, record := range list {
		models = append(models, model.SecretList{
			Id:   int(record.Id),
			Name: record.Name,
		})
	}

	return models, nil
}

func (e *Executor) getSecret(ctx context.Context, args []string) (interface{}, error) {
	switch len(args) - 1 {
	case 0:
		return nil, fmt.Errorf("validation error: Secret ID is missing")
	}

	id, convErr := strconv.Atoi(args[1])
	if convErr != nil {
		return nil, convErr
	}

	secret, err := e.app.DataVaultService.GetDataRecord(ctx, id)
	if err != nil {
		st, _ := status.FromError(err)
		switch st.Code() {
		case codes.NotFound:
			return nil, fmt.Errorf("%s", st.Message())
		default:
			return nil, err
		}
	}

	return secret, nil
}

func (e *Executor) getSecretBinary(ctx context.Context, args []string) error {
	switch len(args) - 1 {
	case 0:
		return fmt.Errorf("validation error: Secret ID and Path is missing")
	case 1:
		return fmt.Errorf("validation error: Path is missing")
	}

	id, errConv := strconv.Atoi(args[1])
	if errConv != nil {
		return errConv
	}

	err := e.app.DataVaultService.GetBinaryDataRecord(ctx, id, args[2])
	if err != nil {
		return err
	}

	return nil
}

func (e *Executor) editSecret(ctx context.Context, args []string, isForce bool) error {
	var (
		recordType int
		id         int
		converted  []byte
	)

	numArgs := len(args) - 1
	if numArgs >= 3 {
		recordType, errConv := strconv.Atoi(args[3])
		if errConv != nil {
			return errConv
		}

		id, errConv = strconv.Atoi(args[1])
		if errConv != nil {
			return errConv
		}

		switch recordType {
		case 1:
			switch numArgs {
			case 4:
				return fmt.Errorf("validation error: Password is missing")
			case 3:
				return fmt.Errorf("validation error: Login and Password is missing")
			default:
				converted, errConv = json.Marshal(model.LoginPassSecret{
					Id:         id,
					Name:       args[2],
					RecordType: 1,
					Login:      args[4],
					Password:   args[5],
				})
				if errConv != nil {
					return errConv
				}
			}
		case 2:
			switch numArgs {
			case 3:
				return fmt.Errorf("validation error: Text is missing")
			default:
				converted, errConv = json.Marshal(model.TextSecret{
					Id:         id,
					Name:       args[2],
					RecordType: 2,
					Text:       strings.Join(args[4:], " "),
				})
				if errConv != nil {
					return errConv
				}
			}
		case 3:
			switch numArgs {
			case 3:
				return fmt.Errorf("validation error: Filepath is missing")
			default:
				converted, errConv = json.Marshal(model.FileSecret{
					Id:         id,
					Name:       args[2],
					RecordType: 3,
					Path:       args[3],
				})
				if errConv != nil {
					return errConv
				}
			}
		case 4:
			switch numArgs {
			case 5:
				return fmt.Errorf("validation error: Due date is missing")
			case 4:
				return fmt.Errorf("validation error: CVV and Due date is missing")
			case 3:
				return fmt.Errorf("validation error: Card number, CVV and Due date is missing")
			default:
				converted, errConv = json.Marshal(model.CardSecret{
					Id:         id,
					Name:       args[2],
					RecordType: 4,
					CardNumber: args[4],
					CVV:        args[5],
					Due:        args[6],
				})
				if errConv != nil {
					return errConv
				}
			}
		}
	} else {
		return fmt.Errorf("validation error: Secret ID, Title, Secret Type ID and secret fields is missing")
	}

	if err := e.app.DataVaultService.UpdateData(ctx, id, args[2], recordType, string(converted), isForce); err != nil {
		st, _ := status.FromError(err)

		fmt.Println(st.Message())

		if st.Code() == codes.FailedPrecondition {
			fmt.Println("starting re-sync")

			e.app.Syncer.SyncAll(ctx)

			fmt.Println("re-sync ended")
		}

		return nil
	}

	return nil
}

func getCommandArgsAndOptions(s string) ([]string, map[string]bool) {
	s = strings.TrimSpace(s)

	setCommand := strings.Split(s, " ")

	l := len(setCommand)

	filtered := make([]string, 0, l)
	options := make(map[string]bool)

	for i := 0; i < len(setCommand); i++ {
		if strings.HasPrefix(setCommand[i], "-") {
			opt := strings.TrimPrefix(setCommand[i], "--")
			opt = strings.TrimPrefix(opt, "-")

			optSplited := strings.Split(opt, "=")
			options[optSplited[0]] = true

			continue
		}
		filtered = append(filtered, setCommand[i])
	}

	return filtered, options
}
