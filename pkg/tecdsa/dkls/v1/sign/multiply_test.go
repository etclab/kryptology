//
// Copyright Coinbase, Inc. All Rights Reserved.
//
// SPDX-License-Identifier: Apache-2.0
//

package sign

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/etclab/kryptology/pkg/core/curves"
	"github.com/etclab/kryptology/pkg/ot/base/simplest"
	"github.com/etclab/kryptology/pkg/ot/extension/kos"
	"github.com/etclab/kryptology/pkg/ot/ottest"
)

func TestMultiply(t *testing.T) {
	curve := curves.K256()
	hashKeySeed := [simplest.DigestSize]byte{}
	_, err := rand.Read(hashKeySeed[:])
	require.NoError(t, err)

	baseOtSenderOutput, baseOtReceiverOutput, err := ottest.RunSimplestOT(curve, kos.Kappa, hashKeySeed)
	require.NoError(t, err)

	sender, err := NewMultiplySender(baseOtReceiverOutput, curve, hashKeySeed)
	require.NoError(t, err)
	receiver, err := NewMultiplyReceiver(baseOtSenderOutput, curve, hashKeySeed)
	require.NoError(t, err)

	alpha := curve.Scalar.Random(rand.Reader)
	beta := curve.Scalar.Random(rand.Reader)

	round1Output, err := receiver.Round1Initialize(beta)
	require.Nil(t, err)
	round2Output, err := sender.Round2Multiply(alpha, round1Output)
	require.Nil(t, err)
	err = receiver.Round3Multiply(round2Output)
	require.Nil(t, err)

	product := alpha.Mul(beta)
	sum := sender.outputAdditiveShare.Add(receiver.outputAdditiveShare)
	require.Equal(t, product, sum)
}
