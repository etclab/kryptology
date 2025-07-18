package v1

import (
	"hash"

	"github.com/pkg/errors"

	"github.com/etclab/kryptology/pkg/core/curves"
	"github.com/etclab/kryptology/pkg/core/protocol"
	"github.com/etclab/kryptology/pkg/tecdsa/dkls/v1/dkg"
	"github.com/etclab/kryptology/pkg/tecdsa/dkls/v1/refresh"
	"github.com/etclab/kryptology/pkg/tecdsa/dkls/v1/sign"
)

// AliceDkg DKLS DKG implementation that satisfies the protocol iterator interface.
type AliceDkg struct {
	protoStepper
	*dkg.Alice
}

// BobDkg DKLS DKG implementation that satisfies the protocol iterator interface.
type BobDkg struct {
	protoStepper
	*dkg.Bob
}

// AliceSign DKLS sign implementation that satisfies the protocol iterator interface.
type AliceSign struct {
	protoStepper
	*sign.Alice
}

// BobSign DKLS sign implementation that satisfies the protocol iterator interface.
type BobSign struct {
	protoStepper
	*sign.Bob
}

// AliceRefresh DKLS refresh implementation that satisfies the protocol iterator interface.
type AliceRefresh struct {
	protoStepper
	*refresh.Alice
}

// BobRefresh DKLS refresh implementation that satisfies the protocol iterator interface.
type BobRefresh struct {
	protoStepper
	*refresh.Bob
}

var (
	// Static type assertions
	_ protocol.Iterator = &AliceDkg{}
	_ protocol.Iterator = &BobDkg{}
	_ protocol.Iterator = &AliceSign{}
	_ protocol.Iterator = &BobSign{}
	_ protocol.Iterator = &AliceRefresh{}
	_ protocol.Iterator = &BobRefresh{}
)

// NewAliceDkg creates a new protocol that can compute a DKG as Alice
func NewAliceDkg(curve *curves.Curve, version uint) *AliceDkg {
	a := &AliceDkg{Alice: dkg.NewAlice(curve)}
	a.steps = []func(*protocol.Message) (*protocol.Message, error){
		func(input *protocol.Message) (*protocol.Message, error) {
			bobSeed, err := decodeDkgRound2Input(input)
			if err != nil {
				return nil, errors.WithStack(err)
			}
			roundOutput, err := a.Round2CommitToProof(bobSeed)
			if err != nil {
				return nil, err
			}
			return encodeDkgRound2Output(roundOutput, version)
		},
		func(input *protocol.Message) (*protocol.Message, error) {
			proof, err := decodeDkgRound4Input(input)
			if err != nil {
				return nil, errors.WithStack(err)
			}
			aliceProof, err := a.Round4VerifyAndReveal(proof)
			if err != nil {
				return nil, err
			}
			return encodeDkgRound4Output(aliceProof, version)
		},
		func(input *protocol.Message) (*protocol.Message, error) {
			proof, err := decodeDkgRound6Input(input)
			if err != nil {
				return nil, errors.WithStack(err)
			}
			choices, err := a.Round6DkgRound2Ot(proof)
			if err != nil {
				return nil, err
			}
			return encodeDkgRound6Output(choices, version)
		},
		func(input *protocol.Message) (*protocol.Message, error) {
			challenge, err := decodeDkgRound8Input(input)
			if err != nil {
				return nil, errors.WithStack(err)
			}
			responses, err := a.Round8DkgRound4Ot(challenge)
			if err != nil {
				return nil, err
			}
			return encodeDkgRound8Output(responses, version)
		},
		func(input *protocol.Message) (*protocol.Message, error) {
			opening, err := decodeDkgRound10Input(input)
			if err != nil {
				return nil, errors.WithStack(err)
			}
			if err := a.Round10DkgRound6Ot(opening); err != nil {
				return nil, err
			}
			return nil, nil
		},
	}
	return a
}

// Result Returns an encoded version of Alice as sequence of bytes that can be used to initialize an AliceSign protocol.
func (a *AliceDkg) Result(version uint) (*protocol.Message, error) {
	// Sanity check
	if !a.complete() {
		return nil, nil
	}
	if a.Alice == nil {
		return nil, protocol.ErrNotInitialized
	}

	result := a.Output()
	return EncodeAliceDkgOutput(result, version)
}

// NewBobDkg Creates a new protocol that can compute a DKG as Bob.
func NewBobDkg(curve *curves.Curve, version uint) *BobDkg {
	b := &BobDkg{Bob: dkg.NewBob(curve)}
	b.steps = []func(message *protocol.Message) (*protocol.Message, error){
		func(*protocol.Message) (*protocol.Message, error) {
			commitment, err := b.Round1GenerateRandomSeed()
			if err != nil {
				return nil, err
			}
			return encodeDkgRound1Output(commitment, version)
		},
		func(input *protocol.Message) (*protocol.Message, error) {
			round3Input, err := decodeDkgRound3Input(input)
			if err != nil {
				return nil, errors.WithStack(err)
			}
			bobProof, err := b.Round3SchnorrProve(round3Input)
			if err != nil {
				return nil, err
			}
			return encodeDkgRound3Output(bobProof, version)
		},
		func(input *protocol.Message) (*protocol.Message, error) {
			proof, err := decodeDkgRound5Input(input)
			if err != nil {
				return nil, errors.WithStack(err)
			}
			bobProof, err := b.Round5DecommitmentAndStartOt(proof)
			if err != nil {
				return nil, err
			}
			return encodeDkgRound5Output(bobProof, version)
		},
		func(input *protocol.Message) (*protocol.Message, error) {
			choices, err := decodeDkgRound7Input(input)
			if err != nil {
				return nil, errors.WithStack(err)
			}
			challenge, err := b.Round7DkgRound3Ot(choices)
			if err != nil {
				return nil, err
			}
			return encodeDkgRound7Output(challenge, version)
		},
		func(input *protocol.Message) (*protocol.Message, error) {
			responses, err := decodeDkgRound9Input(input)
			if err != nil {
				return nil, errors.WithStack(err)
			}
			opening, err := b.Round9DkgRound5Ot(responses)
			if err != nil {
				return nil, err
			}
			return encodeDkgRound9Output(opening, version)
		},
	}
	return b
}

// Result returns an encoded version of Bob as sequence of bytes that can be used to  initialize an BobSign protocol.
func (b *BobDkg) Result(version uint) (*protocol.Message, error) {
	// Sanity check
	if !b.complete() {
		return nil, nil
	}
	if b.Bob == nil {
		return nil, protocol.ErrNotInitialized
	}

	result := b.Output()
	return EncodeBobDkgOutput(result, version)
}

// NewAliceSign creates a new protocol that can compute a signature as Alice.
// Requires dkg state that was produced at the end of DKG.Output().
func NewAliceSign(curve *curves.Curve, hash hash.Hash, message []byte, dkgResultMessage *protocol.Message, version uint) (*AliceSign, error) {
	dkgResult, err := DecodeAliceDkgResult(dkgResultMessage)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	a := &AliceSign{Alice: sign.NewAlice(curve, hash, dkgResult)}
	a.steps = []func(message *protocol.Message) (*protocol.Message, error){
		func(*protocol.Message) (*protocol.Message, error) {
			aliceCommitment, err := a.Round1GenerateRandomSeed()
			if err != nil {
				return nil, err
			}
			return encodeSignRound1Output(aliceCommitment, version)
		},
		func(input *protocol.Message) (*protocol.Message, error) {
			round2Output, err := decodeSignRound3Input(input)
			if err != nil {
				return nil, errors.WithStack(err)
			}
			round3Output, err := a.Round3Sign(message, round2Output)
			if err != nil {
				return nil, errors.WithStack(err)
			}
			return encodeSignRound3Output(round3Output, version)
		},
	}
	return a, nil
}

// NewBobSign creates a new protocol that can compute a signature as Bob.
// Requires dkg state that was produced at the end of DKG.Output().
func NewBobSign(curve *curves.Curve, hash hash.Hash, message []byte, dkgResultMessage *protocol.Message, version uint) (*BobSign, error) {
	dkgResult, err := DecodeBobDkgResult(dkgResultMessage)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	b := &BobSign{Bob: sign.NewBob(curve, hash, dkgResult)}
	b.steps = []func(message *protocol.Message) (*protocol.Message, error){
		func(input *protocol.Message) (*protocol.Message, error) {
			commitment, err := decodeSignRound2Input(input)
			if err != nil {
				return nil, errors.WithStack(err)
			}
			round2Output, err := b.Round2Initialize(commitment)
			if err != nil {
				return nil, errors.WithStack(err)
			}
			return encodeSignRound2Output(round2Output, version)
		},
		func(input *protocol.Message) (*protocol.Message, error) {
			round4Input, err := decodeSignRound4Input(input)
			if err != nil {
				return nil, errors.WithStack(err)
			}

			if err = b.Round4Final(message, round4Input); err != nil {
				return nil, errors.WithStack(err)
			}
			return nil, nil
		},
	}
	return b, nil
}

// Result always returns an error.
// Alice does not compute a signature in the DKLS protocol; only Bob computes the signature.
func (a *AliceSign) Result(_ uint) (*protocol.Message, error) {
	return nil, errors.New("dkls.Alice does not produce a signature")
}

// Result returns the signature that Bob computed as a *core.EcdsaSignature if the signing protocol completed successfully.
func (b *BobSign) Result(version uint) (*protocol.Message, error) {
	// We can't produce a signature until the protocol completes
	if !b.complete() {
		return nil, nil
	}
	if b.Bob == nil {
		// Object wasn't created with NewXSign()
		return nil, protocol.ErrNotInitialized
	}
	return encodeSignature(b.Bob.Signature, version)
}

// NewAliceRefresh creates a new protocol that can compute a key refresh as Alice
func NewAliceRefresh(curve *curves.Curve, dkgResultMessage *protocol.Message, version uint) (*AliceRefresh, error) {
	dkgResult, err := DecodeAliceDkgResult(dkgResultMessage)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	a := &AliceRefresh{Alice: refresh.NewAlice(curve, dkgResult)}
	a.steps = []func(*protocol.Message) (*protocol.Message, error){
		func(input *protocol.Message) (*protocol.Message, error) {
			aliceSeed := a.Round1RefreshGenerateSeed()
			return encodeRefreshRound1Output(aliceSeed, version)
		},
		func(input *protocol.Message) (*protocol.Message, error) {
			round3Input, err := decodeRefreshRound3Input(input)
			if err != nil {
				return nil, errors.WithStack(err)
			}
			round3Output, err := a.Round3RefreshMultiplyRound2Ot(round3Input)
			if err != nil {
				return nil, err
			}
			return encodeRefreshRound3Output(round3Output, version)
		},
		func(input *protocol.Message) (*protocol.Message, error) {
			round5Input, err := decodeRefreshRound5Input(input)
			if err != nil {
				return nil, errors.WithStack(err)
			}
			round5Output, err := a.Round5RefreshRound4Ot(round5Input)
			if err != nil {
				return nil, err
			}
			return encodeRefreshRound5Output(round5Output, version)
		},
		func(input *protocol.Message) (*protocol.Message, error) {
			round7Input, err := decodeRefreshRound7Input(input)
			if err != nil {
				return nil, errors.WithStack(err)
			}
			if err := a.Round7DkgRound6Ot(round7Input); err != nil {
				return nil, err
			}
			return nil, nil
		},
	}
	return a, nil
}

// Result Returns an encoded version of Alice as sequence of bytes that can be used to initialize an AliceSign protocol.
func (a *AliceRefresh) Result(version uint) (*protocol.Message, error) {
	// Sanity check
	if !a.complete() {
		return nil, nil
	}
	if a.Alice == nil {
		return nil, protocol.ErrNotInitialized
	}

	result := a.Output()
	return EncodeAliceDkgOutput(result, version)
}

// NewBobRefresh Creates a new protocol that can compute a refresh as Bob.
func NewBobRefresh(curve *curves.Curve, dkgResultMessage *protocol.Message, version uint) (*BobRefresh, error) {
	dkgResult, err := DecodeBobDkgResult(dkgResultMessage)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	b := &BobRefresh{Bob: refresh.NewBob(curve, dkgResult)}
	b.steps = []func(message *protocol.Message) (*protocol.Message, error){
		func(input *protocol.Message) (*protocol.Message, error) {
			round2Input, err := decodeRefreshRound2Input(input)
			if err != nil {
				return nil, errors.WithStack(err)
			}
			round2Output, err := b.Round2RefreshProduceSeedAndMultiplyAndStartOT(round2Input)
			if err != nil {
				return nil, err
			}
			return encodeRefreshRound2Output(round2Output, version)
		},
		func(input *protocol.Message) (*protocol.Message, error) {
			round4Input, err := decodeRefreshRound4Input(input)
			if err != nil {
				return nil, errors.WithStack(err)
			}
			round4Output, err := b.Round4RefreshRound3Ot(round4Input)
			if err != nil {
				return nil, err
			}
			return encodeRefreshRound4Output(round4Output, version)
		},
		func(input *protocol.Message) (*protocol.Message, error) {
			round6Input, err := decodeRefreshRound6Input(input)
			if err != nil {
				return nil, errors.WithStack(err)
			}
			round6Output, err := b.Round6RefreshRound5Ot(round6Input)
			if err != nil {
				return nil, err
			}
			return encodeRefreshRound6Output(round6Output, version)
		},
	}
	return b, nil
}

// Result returns an encoded version of Bob as sequence of bytes that can be used to  initialize an BobSign protocol.
func (b *BobRefresh) Result(version uint) (*protocol.Message, error) {
	// Sanity check
	if !b.complete() {
		return nil, nil
	}
	if b.Bob == nil {
		return nil, protocol.ErrNotInitialized
	}

	result := b.Output()
	return EncodeBobDkgOutput(result, version)
}
