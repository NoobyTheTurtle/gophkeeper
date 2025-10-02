package storage

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/smanhack/gophkeeper/internal/client/model"
	pb "github.com/smanhack/gophkeeper/pkg/api"
)

type Sync struct {
	storage         Memorier
	dataVaultClient pb.DataVaultClient
	cr              Crypter
}

// NewSync - creates new Sync.
func NewSync(s Memorier, sc pb.DataVaultClient, cr Crypter) *Sync {
	return &Sync{storage: s, dataVaultClient: sc, cr: cr}
}

func (s *Sync) SyncAll(ctx context.Context) {
	if err := s.SyncTextData(ctx); err != nil {
		fmt.Println(err)
	}

	if err := s.SyncPassLoginData(ctx); err != nil {
		fmt.Println(err)
	}

	if err := s.SyncCardData(ctx); err != nil {
		fmt.Println(err)
	}
}

func (s *Sync) SyncTextData(ctx context.Context) error {
	texts, err := s.dataVaultClient.QueryDataByCategory(ctx, &pb.QueryDataByCategoryRequest{TypeId: 2})
	if err != nil {
		panic(err)
	}

	var list []model.TextSecret
	for _, text := range texts.SecretLists {
		id := int(text.Id)
		m := model.TextSecret{}

		decoded, errDecode := s.cr.Decode(string(text.Payload))
		if errDecode != nil {
			return errDecode
		}

		errUnmarshal := json.Unmarshal([]byte(decoded), &m)
		if errUnmarshal != nil {
			return errUnmarshal
		}
		m.Id = id
		m.UpdatedAt = text.UpdatedAt.AsTime()

		list = append(list, m)
	}

	s.storage.SetTextSecrets(list)

	return nil
}

// SyncCardData - makes gRPC request to server and on success sets acquired records to MemoryStorage.CardSecrets.
func (s *Sync) SyncCardData(ctx context.Context) error {
	cards, err := s.dataVaultClient.QueryDataByCategory(ctx, &pb.QueryDataByCategoryRequest{TypeId: 4})
	if err != nil {
		panic(err)
	}

	var list []model.CardSecret
	for _, card := range cards.SecretLists {
		id := int(card.Id)
		m := model.CardSecret{}

		decoded, errDecode := s.cr.Decode(string(card.Payload))
		if errDecode != nil {
			return errDecode
		}

		errUnmarshal := json.Unmarshal([]byte(decoded), &m)
		if errUnmarshal != nil {
			return errUnmarshal
		}
		m.Id = id
		m.UpdatedAt = card.UpdatedAt.AsTime()

		list = append(list, m)
	}

	s.storage.SetCardSecrets(list)

	return nil
}

func (s *Sync) SyncPassLoginData(ctx context.Context) error {
	lists, err := s.dataVaultClient.QueryDataByCategory(ctx, &pb.QueryDataByCategoryRequest{TypeId: 1})
	if err != nil {
		panic(err)
	}

	var list []model.LoginPassSecret
	for _, sList := range lists.SecretLists {
		id := int(sList.Id)
		m := model.LoginPassSecret{}

		decoded, errDecode := s.cr.Decode(string(sList.Payload))
		if errDecode != nil {
			return errDecode
		}

		errUnmarshal := json.Unmarshal([]byte(decoded), &m)
		if errUnmarshal != nil {
			return errUnmarshal
		}
		m.Id = id
		m.UpdatedAt = sList.UpdatedAt.AsTime()

		list = append(list, m)
	}

	s.storage.SetLoginPassSecrets(list)

	return nil
}
