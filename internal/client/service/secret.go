package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/smanhack/gophkeeper/api/proto"
	"github.com/smanhack/gophkeeper/internal/client/model"
	"github.com/smanhack/gophkeeper/internal/client/storage"
	"github.com/smanhack/gophkeeper/pkg/crypt"
)

type DataVaultClientService struct {
	glCtx   *model.GlobalContext
	client  pb.DataVaultClient
	storage storage.Memorier
	crypt   crypt.Crypter
	syncer  storage.Syncer
}

// NewDataVaultClientService - creates new DataVaultClientService.
func NewDataVaultClientService(
	glCtx *model.GlobalContext, client pb.DataVaultClient, st storage.Memorier, cr crypt.Crypter, sr storage.Syncer,
) *DataVaultClientService {
	return &DataVaultClientService{
		glCtx:   glCtx,
		client:  client,
		storage: st,
		crypt:   cr,
		syncer:  sr,
	}
}

func (s *DataVaultClientService) GetListOfDataRecords(id int) ([]*pb.DataRecord, error) {
	list := s.storage.GetDataRecordList(id)
	if len(list) > 0 {
		return list, nil
	}

	result, err := s.client.QueryDataByCategory(s.glCtx.Ctx, &pb.QueryDataByCategoryRequest{TypeId: uint32(id)})
	if err != nil {
		return nil, err
	}

	return result.SecretLists, nil
}

// GetBinaryDataRecord - get binary data from server and stores it into file.
func (s *DataVaultClientService) GetBinaryDataRecord(id int, location string) error {
	res, err := s.client.FetchData(s.glCtx.Ctx, &pb.FetchDataRequest{Id: int32(id)})
	if err != nil {
		return err
	}

	if res.Type != 3 {
		return errors.New("this method only works with binary data, please appropriate method next time")
	}

	decoded, errDecode := s.crypt.Decode(string(res.Payload))
	if errDecode != nil {
		return errDecode
	}

	f, openErr := os.OpenFile(location, os.O_CREATE|os.O_WRONLY, 0o644)
	if openErr != nil {
		return openErr
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			fmt.Printf("Warning: failed to close file: %v\n", closeErr)
		}
	}()

	_, wrError := f.Write([]byte(decoded))
	if wrError != nil {
		return wrError
	}

	return nil
}

// GetDataRecord - attempts to get data record from memory by its id, if nothing is found then makes gRPC request to server.
func (s *DataVaultClientService) GetDataRecord(id int) (interface{}, error) {
	data, ok := s.storage.FindInStorage(id)

	if ok {
		return data, nil
	}

	result, err := s.client.FetchData(s.glCtx.Ctx, &pb.FetchDataRequest{Id: int32(id)})
	if err != nil {
		return nil, err
	}

	if result.Type == 3 {
		return nil, errors.New("to get binary data, pleas use proper method")
	}

	decoded, errDecode := s.crypt.Decode(string(result.Payload))
	if errDecode != nil {
		return nil, errDecode
	}

	var m interface{}
	switch result.Type {
	case 1:
		m = model.LoginPassSecret{}
	case 2:
		m = model.TextSecret{}
	case 4:
		m = model.CardSecret{}
	}

	errUnmarshal := json.Unmarshal([]byte(decoded), &m)
	if errUnmarshal != nil {
		return nil, errUnmarshal
	}

	return m, nil
}

// StoreData - creates new data record on the server and then makes re-sync memory storage.
func (s *DataVaultClientService) StoreData(name string, recordType int, payload string) error {
	payloadT := []byte(s.crypt.Encode(payload))

	result, err := s.client.StoreData(s.glCtx.Ctx, &pb.StoreDataRequest{
		Name:    name,
		Type:    uint32(recordType),
		Payload: payloadT,
	})
	if err != nil {
		return err
	}

	fmt.Println("created new data record with ID:", result.Id)

	s.syncer.SyncAll()

	return nil
}

// RemoveData - removes a data record from server and then makes re-sync memory storage.
func (s *DataVaultClientService) RemoveData(id int) error {
	_, err := s.client.RemoveData(s.glCtx.Ctx, &pb.RemoveDataRequest{Id: uint32(id)})
	if err != nil {
		return err
	}

	fmt.Println("successfully deleted data record")

	s.storage.ResetStorage()
	s.syncer.SyncAll()

	return nil
}

func (s *DataVaultClientService) UpdateData(id int, name string, recordType int, payload string, isForce bool) error {
	var updatedAt time.Time
	localRecord, _ := s.GetDataRecord(id)

	switch record := localRecord.(type) {
	case model.TextSecret:
		updatedAt = record.UpdatedAt
	case model.FileSecret:
		updatedAt = record.UpdatedAt
	case model.LoginPassSecret:
		updatedAt = record.UpdatedAt
	case model.CardSecret:
		updatedAt = record.UpdatedAt
	default:
		panic("unknown type")
	}
	payloadT := []byte(s.crypt.Encode(payload))

	_, err := s.client.UpdateData(
		s.glCtx.Ctx, &pb.UpdateDataRequest{
			Id:        uint32(id),
			Name:      name,
			Type:      uint32(recordType),
			Payload:   payloadT,
			UpdatedAt: timestamppb.New(updatedAt),
			IsForce:   isForce,
		},
	)
	if err != nil {
		return err
	}

	fmt.Println("successfully updated data record")

	s.syncer.SyncAll()

	return nil
}
