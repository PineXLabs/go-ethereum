package engine

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
)

type ExecutionPayloadEnvelopeV2 struct {
	ExecutionPayload *ExecutableData `json:"executionPayload"  gencodec:"required"`
	BlockValue       *big.Int        `json:"blockValue"  gencodec:"required"`
	BlobsBundle      []byte          `json:"blobsBundle" `
	Override         bool            `json:"shouldOverrideBuilder"`
}

func ExecutionPayloadEnvelopeToV2(bd *ExecutionPayloadEnvelope) *ExecutionPayloadEnvelopeV2 {
	bundle := marshalBundle(bd.BlobsBundle)
	res := &ExecutionPayloadEnvelopeV2{
		ExecutionPayload: bd.ExecutionPayload,
		BlockValue:       bd.BlockValue,
		BlobsBundle:      bundle,
		Override:         bd.Override,
	}
	return res
}

func marshalBundle(b *BlobsBundleV1) []byte {
	if len(b.Blobs) == 0 {
		return []byte{}
	}
	start := time.Now()
	bdle := make([]byte, 0, len(b.Blobs)*(4096*32)+len(b.Blobs)*129*48+len(b.Blobs)*48+4)
	bdle = binary.BigEndian.AppendUint32(bdle, uint32(len(b.Blobs)))
	for i := range b.Blobs {
		bdle = append(bdle, b.Blobs[i]...)
	}
	for i := range b.Commitments {
		bdle = append(bdle, b.Commitments[i]...)
	}
	for i := range b.Proofs {
		bdle = append(bdle, b.Proofs[i]...)
	}
	log.Debug("marshalBundle", "result lenght", len(bdle), "bundle blobs", len(b.Blobs), "expected marshal size", len(b.Blobs)*(4096*32)+len(b.Blobs)*129*48+len(b.Blobs)*48+4, "time used", time.Since(start).Milliseconds())
	return bdle
}

func unmarshalBundle(b []byte) (*BlobsBundleV1, error) {
	if len(b) == 0 {
		return &BlobsBundleV1{}, nil
	}
	bdle := b
	length := int(binary.BigEndian.Uint32(bdle[:4]))
	if length > 512 {
		return nil, TooLargeRequest
	}
	if len(bdle) != length*(4096*32)+length*129*48+length*48+4 {
		return nil, errors.New("bad bundle length")
	}
	bdle = bdle[4:]
	ebdle := &BlobsBundleV1{}
	for i := range length {
		ebdle.Blobs = append(ebdle.Blobs, bdle[i*4096*32:(i+1)*4096*32])
	}
	bdle = bdle[(length+1)*4096*32:]
	for i := range length {
		ebdle.Commitments = append(ebdle.Commitments, bdle[i*48:(i+1*48)])
	}
	bdle = bdle[(length+1)*4096*32+(length+1)*48:]
	for i := range length * 129 {
		ebdle.Proofs = append(ebdle.Proofs, bdle[i*48:(i+1*48)])
	}
	return ebdle, nil
}

// MarshalJSON marshals as JSON.
func (e ExecutionPayloadEnvelopeV2) MarshalJSON() ([]byte, error) {
	start := time.Now()
	type ExecutionPayloadEnvelope struct {
		ExecutionPayload *ExecutableData `json:"executionPayload"  gencodec:"required"`
		BlockValue       *hexutil.Big    `json:"blockValue"  gencodec:"required"`
		Override         bool            `json:"shouldOverrideBuilder"`
		BlobsBundle      []byte          `json:"blobsBundle"`
	}
	var enc ExecutionPayloadEnvelope
	enc.ExecutionPayload = e.ExecutionPayload
	enc.BlockValue = (*hexutil.Big)(e.BlockValue)
	enc.BlobsBundle = e.BlobsBundle
	enc.Override = e.Override
	res, err := json.Marshal(&enc)
	if err != nil {
		return nil, err
	}
	log.Debug("marshal ExecutionPayloadEnvelopeV2", "time used", time.Since(start).Milliseconds())
	return res, nil
}

// UnmarshalJSON unmarshals from JSON.
func (e *ExecutionPayloadEnvelopeV2) UnmarshalJSON(input []byte) error {
	type ExecutionPayloadEnvelope struct {
		ExecutionPayload *ExecutableData `json:"executionPayload"  gencodec:"required"`
		BlockValue       *hexutil.Big    `json:"blockValue"  gencodec:"required"`
		Override         *bool           `json:"shouldOverrideBuilder"`
		BlobsBundle      []byte          `json:"blobsBundle"`
	}
	var dec ExecutionPayloadEnvelope
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.ExecutionPayload == nil {
		return errors.New("missing required field 'executionPayload' for ExecutionPayloadEnvelope")
	}
	e.ExecutionPayload = dec.ExecutionPayload
	if dec.BlockValue == nil {
		return errors.New("missing required field 'blockValue' for ExecutionPayloadEnvelope")
	}
	e.BlockValue = (*big.Int)(dec.BlockValue)
	e.BlobsBundle = dec.BlobsBundle
	if dec.Override != nil {
		e.Override = *dec.Override
	}
	return nil
}
