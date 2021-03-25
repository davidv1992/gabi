package rangeproof

import (
	"errors"
	"testing"

	"github.com/privacybydesign/gabi/big"
	"github.com/privacybydesign/gabi/internal/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroup(t *testing.T) {
	g := (qrGroup)(NewQrGroup(big.NewInt(35), big.NewInt(3), big.NewInt(4)))

	assert.Equal(t, g.Base("R"), big.NewInt(3))
	assert.Equal(t, g.Base("S"), big.NewInt(4))
	assert.Equal(t, g.Base("N"), (*big.Int)(nil))
	assert.Equal(t, g.Base("R1234"), (*big.Int)(nil))
	assert.ElementsMatch(t, g.Names(), []string{"R", "S"})

	ret := new(big.Int)
	assert.True(t, g.Exp(ret, "R", big.NewInt(5), g.N))
	assert.Equal(t, ret, new(big.Int).Exp(g.Base("R"), big.NewInt(5), g.N))
	assert.True(t, g.Exp(ret, "S", big.NewInt(7), g.N))
	assert.Equal(t, ret, new(big.Int).Exp(g.Base("S"), big.NewInt(7), g.N))
	assert.False(t, g.Exp(ret, "N", big.NewInt(9), g.N))
	assert.False(t, g.Exp(ret, "R1234", big.NewInt(11), g.N))
}

type bruteForce3 struct{}

func (_ *bruteForce3) Split(delta *big.Int) ([]*big.Int, error) {
	if !delta.IsInt64() {
		panic("too big")
	}

	d := delta.Int64()

	if d > 1e9 || d < 0 {
		panic("too big")
	}

	for i := int64(0); i*i <= d; i++ {
		for j := int64(0); i*i+j*j <= d; j++ {
			for k := int64(0); i*i+j*j+k*k <= d; k++ {
				if i*i+j*j+k*k == d {
					return []*big.Int{big.NewInt(i), big.NewInt(j), big.NewInt(k)}, nil
				}
			}
		}
	}

	panic("Not found")
}

func (_ *bruteForce3) Nsplit() int {
	return 3
}

func (_ *bruteForce3) Ld() uint {
	return 8
}

type bruteForce4 struct{}

func (_ *bruteForce4) Split(delta *big.Int) ([]*big.Int, error) {
	if !delta.IsInt64() {
		panic("too big")
	}

	d := delta.Int64()

	if d > 1e9 || d < 0 {
		panic("too big")
	}

	for i := int64(0); i*i <= d; i++ {
		for j := int64(0); i*i+j*j <= d; j++ {
			for k := int64(0); i*i+j*j+k*k <= d; k++ {
				for l := int64(0); i*i+j*j+k*k+l*l <= d; l++ {
					if i*i+j*j+k*k+l*l == d {
						return []*big.Int{big.NewInt(i), big.NewInt(j), big.NewInt(k), big.NewInt(l)}, nil
					}
				}
			}
		}
	}

	panic("Not found")
}

func (_ *bruteForce4) Nsplit() int {
	return 4
}

func (_ *bruteForce4) Ld() uint {
	return 8
}

func TestRangeProofBasic(t *testing.T) {
	p, ok := new(big.Int).SetString("137638811993558195206420328357073658091105450134788808980204514105755078006531089565424872264423706112211603473814961517434905870865504591672559685691792489986134468104546337570949069664216234978690144943134866212103184925841701142837749906961652202656280177667215409099503103170243548357516953064641207916007", 10)
	require.True(t, ok, "failed to parse p")
	q, ok := new(big.Int).SetString("161568850263671082708797642691138038443080533253276097248590507678645648170870472664501153166861026407778587004276645109302937591955229881186233151561419055453812743980662387119394543989953096207398047305729607795030698835363986813674377580220752360344952636913024495263497458333887018979316817606614095137583", 10)
	require.True(t, ok, "failed to parse q")

	N := new(big.Int).Mul(p, q)

	g := NewQrGroup(N, common.RandomQR(N), common.RandomQR(N))

	s := New(1, big.NewInt(45), &bruteForce3{}, 256, 128, 256)

	m := big.NewInt(112)
	mRandomizer := common.FastRandomBigInt(new(big.Int).Lsh(big.NewInt(1), s.lm+s.lh+s.lstatzk))

	secretList, commit, err := s.CommitmentsFromSecrets(&g, m, mRandomizer)
	require.NoError(t, err)
	proof := s.BuildProof(commit, big.NewInt(1234567))
	assert.True(t, s.VerifyProofStructure(&g, proof))
	proofList := s.CommitmentsFromProof(&g, proof, big.NewInt(1234567))
	assert.Equal(t, secretList, proofList)
}

func TestRangeProofUsingTable(t *testing.T) {
	table := GenerateSquaresTable(65536)

	p, ok := new(big.Int).SetString("137638811993558195206420328357073658091105450134788808980204514105755078006531089565424872264423706112211603473814961517434905870865504591672559685691792489986134468104546337570949069664216234978690144943134866212103184925841701142837749906961652202656280177667215409099503103170243548357516953064641207916007", 10)
	require.True(t, ok, "failed to parse p")
	q, ok := new(big.Int).SetString("161568850263671082708797642691138038443080533253276097248590507678645648170870472664501153166861026407778587004276645109302937591955229881186233151561419055453812743980662387119394543989953096207398047305729607795030698835363986813674377580220752360344952636913024495263497458333887018979316817606614095137583", 10)
	require.True(t, ok, "failed to parse q")

	N := new(big.Int).Mul(p, q)

	g := NewQrGroup(N, common.RandomQR(N), common.RandomQR(N))

	s := New(1, big.NewInt(45), &table, 256, 128, 256)

	m := big.NewInt(112)
	mRandomizer := common.FastRandomBigInt(new(big.Int).Lsh(big.NewInt(1), s.lm+s.lh+s.lstatzk))

	secretList, commit, err := s.CommitmentsFromSecrets(&g, m, mRandomizer)
	require.NoError(t, err)
	proof := s.BuildProof(commit, big.NewInt(1234567))
	assert.True(t, s.VerifyProofStructure(&g, proof))
	proofList := s.CommitmentsFromProof(&g, proof, big.NewInt(1234567))
	assert.Equal(t, secretList, proofList)
}

func TestRangeProofUsingSumFourSquareAlg(t *testing.T) {
	p, ok := new(big.Int).SetString("137638811993558195206420328357073658091105450134788808980204514105755078006531089565424872264423706112211603473814961517434905870865504591672559685691792489986134468104546337570949069664216234978690144943134866212103184925841701142837749906961652202656280177667215409099503103170243548357516953064641207916007", 10)
	require.True(t, ok, "failed to parse p")
	q, ok := new(big.Int).SetString("161568850263671082708797642691138038443080533253276097248590507678645648170870472664501153166861026407778587004276645109302937591955229881186233151561419055453812743980662387119394543989953096207398047305729607795030698835363986813674377580220752360344952636913024495263497458333887018979316817606614095137583", 10)
	require.True(t, ok, "failed to parse q")

	N := new(big.Int).Mul(p, q)

	g := NewQrGroup(N, common.RandomQR(N), common.RandomQR(N))

	s := New(1, big.NewInt(45), &FourSquareSplitter{}, 256, 128, 256)

	m := big.NewInt(112)
	mRandomizer := common.FastRandomBigInt(new(big.Int).Lsh(big.NewInt(1), s.lm+s.lh+s.lstatzk))

	secretList, commit, err := s.CommitmentsFromSecrets(&g, m, mRandomizer)
	require.NoError(t, err)
	proof := s.BuildProof(commit, big.NewInt(1234567))
	assert.True(t, s.VerifyProofStructure(&g, proof))
	proofList := s.CommitmentsFromProof(&g, proof, big.NewInt(1234567))
	assert.Equal(t, secretList, proofList)
}

func TestRangeProofInvalidStatement(t *testing.T) {
	p, ok := new(big.Int).SetString("137638811993558195206420328357073658091105450134788808980204514105755078006531089565424872264423706112211603473814961517434905870865504591672559685691792489986134468104546337570949069664216234978690144943134866212103184925841701142837749906961652202656280177667215409099503103170243548357516953064641207916007", 10)
	require.True(t, ok, "failed to parse p")
	q, ok := new(big.Int).SetString("161568850263671082708797642691138038443080533253276097248590507678645648170870472664501153166861026407778587004276645109302937591955229881186233151561419055453812743980662387119394543989953096207398047305729607795030698835363986813674377580220752360344952636913024495263497458333887018979316817606614095137583", 10)
	require.True(t, ok, "failed to parse q")

	N := new(big.Int).Mul(p, q)

	g := NewQrGroup(N, common.RandomQR(N), common.RandomQR(N))

	s := New(1, big.NewInt(113), &bruteForce3{}, 256, 128, 256)

	m := big.NewInt(112)
	mRandomizer := common.FastRandomBigInt(new(big.Int).Lsh(big.NewInt(1), s.lm+s.lh+s.lstatzk))

	_, _, err := s.CommitmentsFromSecrets(&g, m, mRandomizer)
	assert.Error(t, err)
}

func TestRangeProofBasic4(t *testing.T) {
	p, ok := new(big.Int).SetString("137638811993558195206420328357073658091105450134788808980204514105755078006531089565424872264423706112211603473814961517434905870865504591672559685691792489986134468104546337570949069664216234978690144943134866212103184925841701142837749906961652202656280177667215409099503103170243548357516953064641207916007", 10)
	require.True(t, ok, "failed to parse p")
	q, ok := new(big.Int).SetString("161568850263671082708797642691138038443080533253276097248590507678645648170870472664501153166861026407778587004276645109302937591955229881186233151561419055453812743980662387119394543989953096207398047305729607795030698835363986813674377580220752360344952636913024495263497458333887018979316817606614095137583", 10)
	require.True(t, ok, "failed to parse q")

	N := new(big.Int).Mul(p, q)

	g := NewQrGroup(N, common.RandomQR(N), common.RandomQR(N))

	s := New(1, big.NewInt(45), &bruteForce4{}, 256, 128, 256)

	m := big.NewInt(112)
	mRandomizer := common.FastRandomBigInt(new(big.Int).Lsh(big.NewInt(1), s.lm+s.lh+s.lstatzk))

	secretList, commit, err := s.CommitmentsFromSecrets(&g, m, mRandomizer)
	require.NoError(t, err)
	proof := s.BuildProof(commit, big.NewInt(1234567))
	assert.True(t, s.VerifyProofStructure(&g, proof))
	proofList := s.CommitmentsFromProof(&g, proof, big.NewInt(1234567))
	assert.Equal(t, secretList, proofList)
}

type testSplit struct {
	val []*big.Int
	e   error
	n   int
	ld  uint
}

func (t *testSplit) Split(_ *big.Int) ([]*big.Int, error) {
	return t.val, t.e
}

func (t *testSplit) Nsplit() int {
	return t.n
}

func (t *testSplit) Ld() uint {
	return t.ld
}

func TestRangeProofMisbehavingSplit(t *testing.T) {
	p, ok := new(big.Int).SetString("137638811993558195206420328357073658091105450134788808980204514105755078006531089565424872264423706112211603473814961517434905870865504591672559685691792489986134468104546337570949069664216234978690144943134866212103184925841701142837749906961652202656280177667215409099503103170243548357516953064641207916007", 10)
	require.True(t, ok, "failed to parse p")
	q, ok := new(big.Int).SetString("161568850263671082708797642691138038443080533253276097248590507678645648170870472664501153166861026407778587004276645109302937591955229881186233151561419055453812743980662387119394543989953096207398047305729607795030698835363986813674377580220752360344952636913024495263497458333887018979316817606614095137583", 10)
	require.True(t, ok, "failed to parse q")

	N := new(big.Int).Mul(p, q)

	g := NewQrGroup(N, common.RandomQR(N), common.RandomQR(N))

	s := New(1, big.NewInt(45), &testSplit{val: nil, e: errors.New("test"), n: 4, ld: 8}, 256, 128, 256)

	m := big.NewInt(112)
	mRandomizer := common.FastRandomBigInt(new(big.Int).Lsh(big.NewInt(1), s.lm+s.lh+s.lstatzk))

	_, _, err := s.CommitmentsFromSecrets(&g, m, mRandomizer)
	assert.Error(t, err)

	s = New(1, big.NewInt(45), &testSplit{val: []*big.Int{big.NewInt(512), big.NewInt(512), big.NewInt(512)}, e: nil, n: 3, ld: 8}, 256, 128, 256)
	_, _, err = s.CommitmentsFromSecrets(&g, m, mRandomizer)
	assert.Error(t, err)

	s = New(1, big.NewInt(45), &testSplit{val: []*big.Int{big.NewInt(1), big.NewInt(1), big.NewInt(1)}, e: nil, n: 4, ld: 8}, 256, 128, 256)
	_, _, err = s.CommitmentsFromSecrets(&g, m, mRandomizer)
	assert.Error(t, err)

	s = New(1, big.NewInt(45), &testSplit{val: []*big.Int{big.NewInt(1), big.NewInt(1), big.NewInt(1)}, e: nil, n: 3, ld: 8}, 256, 128, 256)
	secretList, commit, err := s.CommitmentsFromSecrets(&g, m, mRandomizer)
	require.NoError(t, err)
	proof := s.BuildProof(commit, big.NewInt(1234567))
	assert.True(t, s.VerifyProofStructure(&g, proof))
	proofList := s.CommitmentsFromProof(&g, proof, big.NewInt(1234567))
	assert.NotEqual(t, secretList, proofList)
}

func TestProofKeyproofInterfaces(t *testing.T) {
	p, ok := new(big.Int).SetString("137638811993558195206420328357073658091105450134788808980204514105755078006531089565424872264423706112211603473814961517434905870865504591672559685691792489986134468104546337570949069664216234978690144943134866212103184925841701142837749906961652202656280177667215409099503103170243548357516953064641207916007", 10)
	require.True(t, ok, "failed to parse p")
	q, ok := new(big.Int).SetString("161568850263671082708797642691138038443080533253276097248590507678645648170870472664501153166861026407778587004276645109302937591955229881186233151561419055453812743980662387119394543989953096207398047305729607795030698835363986813674377580220752360344952636913024495263497458333887018979316817606614095137583", 10)
	require.True(t, ok, "failed to parse q")

	N := new(big.Int).Mul(p, q)

	g := NewQrGroup(N, common.RandomQR(N), common.RandomQR(N))

	s := New(1, big.NewInt(45), &bruteForce3{}, 256, 128, 256)

	m := big.NewInt(112)
	mRandomizer := common.FastRandomBigInt(new(big.Int).Lsh(big.NewInt(1), s.lm+s.lh+s.lstatzk))

	_, commit, err := s.CommitmentsFromSecrets(&g, m, mRandomizer)
	require.NoError(t, err)
	proof := (*proof)(s.BuildProof(commit, big.NewInt(1234567)))

	assert.ElementsMatch(t, proof.Names(), []string{"C0", "C1", "C2"})
	assert.Equal(t, proof.Base("C0"), proof.C[0])
	assert.Equal(t, proof.Base("C1"), proof.C[1])
	assert.Equal(t, proof.Base("C2"), proof.C[2])
	assert.Equal(t, proof.Base("C-1"), (*big.Int)(nil))
	assert.Equal(t, proof.Base("C3"), (*big.Int)(nil))
	assert.Equal(t, proof.Base("Cabcd"), (*big.Int)(nil))
	assert.Equal(t, proof.Base("djfdf"), (*big.Int)(nil))

	ret := new(big.Int)
	assert.True(t, proof.Exp(ret, "C0", big.NewInt(15), g.N))
	assert.Equal(t, ret, new(big.Int).Exp(proof.Base("C0"), big.NewInt(15), g.N))
	assert.True(t, proof.Exp(ret, "C1", big.NewInt(17), g.N))
	assert.Equal(t, ret, new(big.Int).Exp(proof.Base("C1"), big.NewInt(17), g.N))
	assert.True(t, proof.Exp(ret, "C2", big.NewInt(19), g.N))
	assert.Equal(t, ret, new(big.Int).Exp(proof.Base("C2"), big.NewInt(19), g.N))
	assert.False(t, proof.Exp(ret, "C3", big.NewInt(21), g.N))
	assert.False(t, proof.Exp(ret, "C-1", big.NewInt(23), g.N))
	assert.False(t, proof.Exp(ret, "Cadfsdf", big.NewInt(25), g.N))
	assert.False(t, proof.Exp(ret, "jdsdfj", big.NewInt(27), g.N))

	assert.Equal(t, proof.ProofResult("v0"), proof.VResponse[0])
	assert.Equal(t, proof.ProofResult("v1"), proof.VResponse[1])
	assert.Equal(t, proof.ProofResult("v2"), proof.VResponse[2])
	assert.Equal(t, proof.ProofResult("v-1"), (*big.Int)(nil))
	assert.Equal(t, proof.ProofResult("v3"), (*big.Int)(nil))
	assert.Equal(t, proof.ProofResult("vajdfsk"), (*big.Int)(nil))
	assert.Equal(t, proof.ProofResult("d0"), proof.DResponse[0])
	assert.Equal(t, proof.ProofResult("d1"), proof.DResponse[1])
	assert.Equal(t, proof.ProofResult("d2"), proof.DResponse[2])
	assert.Equal(t, proof.ProofResult("d-1"), (*big.Int)(nil))
	assert.Equal(t, proof.ProofResult("d3"), (*big.Int)(nil))
	assert.Equal(t, proof.ProofResult("dalsdf"), (*big.Int)(nil))
	assert.Equal(t, proof.ProofResult("m"), proof.MResponse)
	assert.Equal(t, proof.ProofResult("msdfjk"), (*big.Int)(nil))
	assert.Equal(t, proof.ProofResult("sjfd"), (*big.Int)(nil))

}

func TestCommitKeyproofInterfaces(t *testing.T) {
	p, ok := new(big.Int).SetString("137638811993558195206420328357073658091105450134788808980204514105755078006531089565424872264423706112211603473814961517434905870865504591672559685691792489986134468104546337570949069664216234978690144943134866212103184925841701142837749906961652202656280177667215409099503103170243548357516953064641207916007", 10)
	require.True(t, ok, "failed to parse p")
	q, ok := new(big.Int).SetString("161568850263671082708797642691138038443080533253276097248590507678645648170870472664501153166861026407778587004276645109302937591955229881186233151561419055453812743980662387119394543989953096207398047305729607795030698835363986813674377580220752360344952636913024495263497458333887018979316817606614095137583", 10)
	require.True(t, ok, "failed to parse q")

	N := new(big.Int).Mul(p, q)

	g := NewQrGroup(N, common.RandomQR(N), common.RandomQR(N))

	s := New(1, big.NewInt(45), &bruteForce3{}, 256, 128, 256)

	m := big.NewInt(112)
	mRandomizer := common.FastRandomBigInt(new(big.Int).Lsh(big.NewInt(1), s.lm+s.lh+s.lstatzk))

	_, commit_, err := s.CommitmentsFromSecrets(&g, m, mRandomizer)
	require.NoError(t, err)

	commit := (*proofCommit)(commit_)

	assert.ElementsMatch(t, commit.Names(), []string{"C0", "C1", "C2"})
	assert.Equal(t, commit.Base("C0"), commit.c[0])
	assert.Equal(t, commit.Base("C1"), commit.c[1])
	assert.Equal(t, commit.Base("C2"), commit.c[2])
	assert.Equal(t, commit.Base("C3"), (*big.Int)(nil))
	assert.Equal(t, commit.Base("C-1"), (*big.Int)(nil))
	assert.Equal(t, commit.Base("Cadfsdf"), (*big.Int)(nil))
	assert.Equal(t, commit.Base("jdsdfj"), (*big.Int)(nil))

	ret := new(big.Int)
	assert.True(t, commit.Exp(ret, "C0", big.NewInt(15), g.N))
	assert.Equal(t, ret, new(big.Int).Exp(commit.Base("C0"), big.NewInt(15), g.N))
	assert.True(t, commit.Exp(ret, "C1", big.NewInt(17), g.N))
	assert.Equal(t, ret, new(big.Int).Exp(commit.Base("C1"), big.NewInt(17), g.N))
	assert.True(t, commit.Exp(ret, "C2", big.NewInt(19), g.N))
	assert.Equal(t, ret, new(big.Int).Exp(commit.Base("C2"), big.NewInt(19), g.N))
	assert.False(t, commit.Exp(ret, "C3", big.NewInt(21), g.N))
	assert.False(t, commit.Exp(ret, "C-1", big.NewInt(23), g.N))
	assert.False(t, commit.Exp(ret, "Cadfsdf", big.NewInt(25), g.N))
	assert.False(t, commit.Exp(ret, "jdsdfj", big.NewInt(27), g.N))

	assert.Equal(t, commit.Secret("v5"), commit.v5)
	assert.Equal(t, commit.Secret("v0"), commit.v[0])
	assert.Equal(t, commit.Secret("v1"), commit.v[1])
	assert.Equal(t, commit.Secret("v2"), commit.v[2])
	assert.Equal(t, commit.Secret("v3"), (*big.Int)(nil))
	assert.Equal(t, commit.Secret("v-1"), (*big.Int)(nil))
	assert.Equal(t, commit.Secret("vasjdfl"), (*big.Int)(nil))
	assert.Equal(t, commit.Secret("d0"), commit.d[0])
	assert.Equal(t, commit.Secret("d1"), commit.d[1])
	assert.Equal(t, commit.Secret("d2"), commit.d[2])
	assert.Equal(t, commit.Secret("d-1"), (*big.Int)(nil))
	assert.Equal(t, commit.Secret("d3"), (*big.Int)(nil))
	assert.Equal(t, commit.Secret("dasdlkfj"), (*big.Int)(nil))
	assert.Equal(t, commit.Secret("m"), m)
	assert.Equal(t, commit.Secret("malsd"), (*big.Int)(nil))
	assert.Equal(t, commit.Secret("alsdkjf"), (*big.Int)(nil))

	assert.Equal(t, commit.Randomizer("v5"), commit.v5Randomizer)
	assert.Equal(t, commit.Randomizer("v0"), commit.vRandomizer[0])
	assert.Equal(t, commit.Randomizer("v1"), commit.vRandomizer[1])
	assert.Equal(t, commit.Randomizer("v2"), commit.vRandomizer[2])
	assert.Equal(t, commit.Randomizer("v3"), (*big.Int)(nil))
	assert.Equal(t, commit.Randomizer("v-1"), (*big.Int)(nil))
	assert.Equal(t, commit.Randomizer("vasjdfl"), (*big.Int)(nil))
	assert.Equal(t, commit.Randomizer("d0"), commit.dRandomizer[0])
	assert.Equal(t, commit.Randomizer("d1"), commit.dRandomizer[1])
	assert.Equal(t, commit.Randomizer("d2"), commit.dRandomizer[2])
	assert.Equal(t, commit.Randomizer("d-1"), (*big.Int)(nil))
	assert.Equal(t, commit.Randomizer("d3"), (*big.Int)(nil))
	assert.Equal(t, commit.Randomizer("dasdlkfj"), (*big.Int)(nil))
	assert.Equal(t, commit.Randomizer("m"), mRandomizer)
	assert.Equal(t, commit.Randomizer("malsd"), (*big.Int)(nil))
	assert.Equal(t, commit.Randomizer("alsdkjf"), (*big.Int)(nil))
}

func TestVerifyProofStructure(t *testing.T) {
	p, ok := new(big.Int).SetString("137638811993558195206420328357073658091105450134788808980204514105755078006531089565424872264423706112211603473814961517434905870865504591672559685691792489986134468104546337570949069664216234978690144943134866212103184925841701142837749906961652202656280177667215409099503103170243548357516953064641207916007", 10)
	require.True(t, ok, "failed to parse p")
	q, ok := new(big.Int).SetString("161568850263671082708797642691138038443080533253276097248590507678645648170870472664501153166861026407778587004276645109302937591955229881186233151561419055453812743980662387119394543989953096207398047305729607795030698835363986813674377580220752360344952636913024495263497458333887018979316817606614095137583", 10)
	require.True(t, ok, "failed to parse q")

	N := new(big.Int).Mul(p, q)

	g := NewQrGroup(N, common.RandomQR(N), common.RandomQR(N))

	s := New(1, big.NewInt(45), &bruteForce3{}, 256, 128, 256)

	m := big.NewInt(112)
	mRandomizer := common.FastRandomBigInt(new(big.Int).Lsh(big.NewInt(1), s.lm+s.lh+s.lstatzk))

	_, commit, err := s.CommitmentsFromSecrets(&g, m, mRandomizer)
	require.NoError(t, err)
	proof := s.BuildProof(commit, big.NewInt(1234567))

	backup := new(big.Int).Set(proof.MResponse)
	proof.MResponse.Lsh(proof.MResponse, 2049)
	assert.False(t, s.VerifyProofStructure(&g, proof))
	proof.MResponse = nil
	assert.False(t, s.VerifyProofStructure(&g, proof))
	proof.MResponse = backup
	assert.True(t, s.VerifyProofStructure(&g, proof))

	backup = new(big.Int).Set(proof.VCombinedResponse)
	proof.VCombinedResponse.Lsh(proof.VCombinedResponse, 2049)
	assert.False(t, s.VerifyProofStructure(&g, proof))
	proof.VCombinedResponse = nil
	assert.False(t, s.VerifyProofStructure(&g, proof))
	proof.VCombinedResponse = backup
	assert.True(t, s.VerifyProofStructure(&g, proof))

	for i := range proof.C {
		backup = new(big.Int).Set(proof.C[i])
		proof.C[i].Lsh(proof.C[i], 2049)
		assert.False(t, s.VerifyProofStructure(&g, proof))
		proof.C[i] = nil
		assert.False(t, s.VerifyProofStructure(&g, proof))
		proof.C[i] = backup
		assert.True(t, s.VerifyProofStructure(&g, proof))
	}

	for i := range proof.DResponse {
		backup = new(big.Int).Set(proof.DResponse[i])
		proof.DResponse[i].Lsh(proof.DResponse[i], 2049)
		assert.False(t, s.VerifyProofStructure(&g, proof))
		proof.DResponse[i] = nil
		assert.False(t, s.VerifyProofStructure(&g, proof))
		proof.DResponse[i] = backup
		assert.True(t, s.VerifyProofStructure(&g, proof))
	}

	for i := range proof.VResponse {
		backup = new(big.Int).Set(proof.VResponse[i])
		proof.VResponse[i].Lsh(proof.VResponse[i], 2049)
		assert.False(t, s.VerifyProofStructure(&g, proof))
		proof.VResponse[i] = nil
		assert.False(t, s.VerifyProofStructure(&g, proof))
		proof.VResponse[i] = backup
		assert.True(t, s.VerifyProofStructure(&g, proof))
	}

	backup = new(big.Int).Set(proof.C[len(proof.C)-1])
	proof.C = append(proof.C, big.NewInt(15))
	assert.False(t, s.VerifyProofStructure(&g, proof))
	proof.C = proof.C[:len(proof.C)-2]
	assert.False(t, s.VerifyProofStructure(&g, proof))
	proof.C = append(proof.C, backup)
	assert.True(t, s.VerifyProofStructure(&g, proof))

	backup = new(big.Int).Set(proof.DResponse[len(proof.DResponse)-1])
	proof.DResponse = append(proof.DResponse, big.NewInt(15))
	assert.False(t, s.VerifyProofStructure(&g, proof))
	proof.DResponse = proof.DResponse[:len(proof.DResponse)-2]
	assert.False(t, s.VerifyProofStructure(&g, proof))
	proof.DResponse = append(proof.DResponse, backup)
	assert.True(t, s.VerifyProofStructure(&g, proof))

	backup = new(big.Int).Set(proof.VResponse[len(proof.VResponse)-1])
	proof.VResponse = append(proof.VResponse, big.NewInt(15))
	assert.False(t, s.VerifyProofStructure(&g, proof))
	proof.VResponse = proof.VResponse[:len(proof.VResponse)-2]
	assert.False(t, s.VerifyProofStructure(&g, proof))
	proof.VResponse = append(proof.VResponse, backup)
	assert.True(t, s.VerifyProofStructure(&g, proof))
}
