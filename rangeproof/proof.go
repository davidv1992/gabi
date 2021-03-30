package rangeproof

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/privacybydesign/gabi/big"
	"github.com/privacybydesign/gabi/internal/common"
	"github.com/privacybydesign/gabi/keyproof"
)

// This subpackage of gabi implements a variation of the inequality/range proof protocol given in section 6.2.6/6.3.6 of "
// Specification of the Identity Mixer Cryptographic Library
// Version 2.3.0"
//
// The following changes were made:
//  - There is no direct support for the > and < operators, the end user should do boundary adjustment for this themselves
//  - There is support for sum of 3 squares as well as sum of 4 squares for proving the delta is positive
//  - There is no separate commitment to the difference between bound and attribute value.
//
// This results in that our code proves the following substatement:
// C_i = R^d_i S^v_i
// R^k \product_i C_i^d_i = R^a*m S^v_5
//
// where k, a are fixed constants specified in the proof structure
// d_i are values such that a*m-k = \sum_i d_i^2
// v_i are computational hiders for the d_i
// v_5 = \sum_i d_i *v_i
// m is the attribute value
//
//
// The proof of soundness for this protocol is relatively straightforward, but as we are not aware of its occurence in literature,
// we provide it here for completeness:
// ----
//
// First note that we can assume a=1 without loss of generality. Then we define:
//
// Adversary A: Turing machine taking as starting input S, R, Z, n, and k, then participates as receiver in an issuance protocol to
// obtain a CL signature on m < k, then participates as prover in a proving protocol showing a CL-signature on m', as well as
// providing bases C1, C2, C3 and proving knowledge of d1, d2, d3, v1, v2, v3, v5 such that
// C1 = R^d1 S^v1
// C2 = R^d2 S^v2
// C3 = R^d3 S^v3
// C1^d1 C2^d2 C3^d3 R^k = R^m' S^v5
// with succes probability at least epsilon, where epsilon is a non-neglible function of log(n).
//
// Theorem:
// Existence of adversary A contradicts the strong RSA assumption.
//
// Proof:
//
// From A we can derive two turing machines F and G
//
// F:
// Run A, then rewind to extract the full CL signature on m'. Then return that signature if m != m', fail otherwise
//
// G:
// Run A, then rewind to extract m', d1, d2, d3, v1, v2, v3, v5. Then return m, d1, d2, d3, v1, v2, v3, v5 iff m == m', fail otherwise
//
// By construction at least one of F, G will succeed with probability at least epsilon/2
//
// By theorem 1 of "A Signature Scheme with Efficient Protocols" by Camenisch and Lysyanskaya (CL03), existence of F with non-neglible
// succes probability contradicts the strong RSA assumption.
//
// Next, let us show that existence of G with non-neglible succes probability also contradicts the strong RSA assumption. Let (n, u)
// be a flexible RSA problem. We choose random prime e > 4, random integers r1, r2, k, and m>k, and random element v in QRn. We then let
// S = u^e
// R = S^r1
// Z = S^r2
// and present S,R,Z,n as public key to the adversary, together with k.
//
// We then use our knowledge of the e-th roots of S, R, and Z, together with our ability to extract v' from G to issue the signature on
// m similar to the approach taken in the proofs of lemma's 3-5 in CL03
// Next, we provide k and receive back m, d1, d2, d3, v1, v2, v3 and v5, for which it holds that
// R^(d1^2+d2^2+d3^2+k-m) = S^(v5-d1*v1+d2*v2+d3*v3).
//
// Since k > m, and since d1, d2, d3 are real, we have d1^2+d2^2+d3^2+k-m > 0. Now either phi(n) divides d1^2+d2^2+d3^2+k-m, or
// v5-d1*v1+d2*v2+d3*v3 is not zero. If phi(n) divides d1^2+d2^2+d3^2+k-m, then by Lemma 11 in CL03, we can factor n and trivially
// solve the instance.
//
// Otherwise, we now have nonzero a = d1^2+d2^2+d3^2+k-m, and b = v5-d1*v1+d2*v2+d3*v3, such that R^a = S^b. Since e does not divide phi(n)
// (otherwise, we could just factor n again, this time by lemma 12 in CL03), this also implies u^(r1*a) = u^b. Since a and b are bounded
// (each of their components is proven to be smaller than a bound during the ZKP), we can take r1 large enough to guarantee r1*a > b.
// Then r1*a-b > 0, and by multiplication by u^-b, we get u^(r1*a-b) = 1, hence phi(n) | r1*a-b, which means we can factor n and use
// that to solve the flexible RSA problem.
//
// This shows existence of G with non-neglible succes probability also contradicts strong RSA, hence existence of A contradicts
// the strong RSA assumption.
//
// The techniques used to fake issuance can be generalized to multiple issuances using techiques similar to those used in CL03 for lemmas 3-5,
// the rest of the proof then needs e to be replaced with E, the product of all used e_i's.

type (
	ProofStructure struct {
		cRep     []qrRepresentationProofStructure
		mCorrect qrRepresentationProofStructure

		a int
		k *big.Int

		splitter SquareSplitter
		ld       uint
		lm       uint
		lh       uint
		lstatzk  uint
	}

	Proof struct {
		// Actual proof responses
		Cs         []*big.Int `json:"C"`
		DResponses []*big.Int `json:"d"`
		VResponses []*big.Int `json:"v"`
		V5Response *big.Int   `json:"v5"`
		MResponse  *big.Int   `json:"m"`

		// Proof structure description
		Ld uint     `json:"l_d"`
		A  int      `json:"a"`
		K  *big.Int `json:"k"`
	}

	ProofCommit struct {
		// Bases
		c []*big.Int

		// Secrets
		d            []*big.Int
		dRandomizers []*big.Int
		v            []*big.Int
		vRandomizers []*big.Int
		v5           *big.Int
		v5Randomizer *big.Int
		m            *big.Int
		mRandomizer  *big.Int
	}

	proof       Proof
	proofCommit ProofCommit
	qrGroup     QrGroup
)

// Create a new proof structure for proving a statement of form a*m - k >= 0
//  splitter describes the method used for splitting numbers into sum of squares
//  lh is the size of the challenge
//  lm the size of m, and also used as the number of bits for computational hiding
//  lstatzk the number of bits of statistical hiding to use
func New(a int, k *big.Int, split SquareSplitter, lh, lstatzk, lm uint) *ProofStructure {
	if split.SquareCount() == 3 {
		// Not all numbers can be written as sum of 3 squares, but n for which n == 2 (mod 4) can
		// so ensure that a*m-k falls into that category
		a *= 4
		k = new(big.Int).Mul(k, big.NewInt(4)) // ensure we dont overwrite callers copy of k
		k.Sub(k, big.NewInt(2))
	}

	return newWithParams(a, k, split, split.SquareCount(), split.Ld(), lh, lstatzk, lm)
}

func newWithParams(a int, k *big.Int, split SquareSplitter, nSplit int, ld, lh, lstatzk, lm uint) *ProofStructure {
	if nSplit > 4 {
		panic("No support for range proofs with delta split in more than 4 squares")
	}

	result := &ProofStructure{
		mCorrect: qrRepresentationProofStructure{
			Lhs: []keyproof.LhsContribution{
				{Base: "R", Power: new(big.Int).Neg(k)},
			},
			Rhs: []keyproof.RhsContribution{
				{Base: "S", Secret: "v5", Power: -1},
				{Base: "R", Secret: "m", Power: int64(-a)},
			},
		},

		a: a,
		k: new(big.Int).Set(k),

		splitter: split,
		ld:       ld,
		lm:       lm,
		lh:       lh,
		lstatzk:  lstatzk,
	}

	for i := 0; i < nSplit; i++ {
		result.cRep = append(result.cRep, qrRepresentationProofStructure{
			Lhs: []keyproof.LhsContribution{
				{Base: fmt.Sprintf("C%d", i), Power: big.NewInt(1)},
			},
			Rhs: []keyproof.RhsContribution{
				{Base: "R", Secret: fmt.Sprintf("d%d", i), Power: 1},
				{Base: "S", Secret: fmt.Sprintf("v%d", i), Power: 1},
			},
		})

		result.mCorrect.Rhs = append(result.mCorrect.Rhs, keyproof.RhsContribution{
			Base:   fmt.Sprintf("C%d", i),
			Secret: fmt.Sprintf("d%d", i),
			Power:  1,
		})
	}

	return result
}

func (s *ProofStructure) CommitmentsFromSecrets(g *QrGroup, m, mRandomizer *big.Int) ([]*big.Int, *ProofCommit, error) {
	var err error

	d := new(big.Int).Mul(m, big.NewInt(int64(s.a)))
	d.Sub(d, s.k)

	if d.Sign() < 0 {
		return nil, nil, errors.New("Requested inequality does not hold")
	}

	commit := &proofCommit{
		m:           m,
		mRandomizer: mRandomizer,
	}

	commit.d, err = s.splitter.Split(d)
	if err != nil {
		return nil, nil, err
	}
	if len(commit.d) != len(s.cRep) {
		return nil, nil, errors.New("Split function returned wrong number of results")
	}

	// Check d values and generate randomizers for them
	commit.dRandomizers = make([]*big.Int, len(commit.d))
	for i, v := range commit.d {
		if v.BitLen() > int(s.ld) {
			return nil, nil, errors.New("Split function returned oversized d")
		}
		commit.dRandomizers[i], err = common.RandomBigInt(s.ld + s.lh + s.lstatzk)
		if err != nil {
			return nil, nil, err
		}
	}

	// Generate v and vRandomizers
	commit.v = make([]*big.Int, len(commit.d))
	commit.vRandomizers = make([]*big.Int, len(commit.d))
	for i := range commit.d {
		commit.v[i], err = common.RandomBigInt(s.lm)
		if err != nil {
			return nil, nil, err
		}
		commit.vRandomizers[i], err = common.RandomBigInt(s.lm + s.lh + s.lstatzk)
		if err != nil {
			return nil, nil, err
		}
	}

	// Generate v5 and its randomizer
	commit.v5 = big.NewInt(0)
	for i := range commit.d {
		contrib := new(big.Int).Mul(commit.d[i], commit.v[i])
		commit.v5.Add(commit.v5, contrib)
	}
	commit.v5Randomizer, err = common.RandomBigInt(s.lm + s.ld + 2 + s.lh + s.lstatzk)
	if err != nil {
		return nil, nil, err
	}

	// Calculate the bases
	commit.c = make([]*big.Int, len(commit.d))
	for i := range commit.d {
		commit.c[i] = new(big.Int).Exp(g.R, commit.d[i], g.N)
		commit.c[i].Mul(commit.c[i], new(big.Int).Exp(g.S, commit.v[i], g.N))
		commit.c[i].Mod(commit.c[i], g.N)
	}

	bases := keyproof.NewBaseMerge((*qrGroup)(g), commit)

	contributions := []*big.Int{}
	contributions = s.mCorrect.commitmentsFromSecrets((*qrGroup)(g), contributions, &bases, commit)
	for i := range commit.d {
		contributions = s.cRep[i].commitmentsFromSecrets((*qrGroup)(g), contributions, &bases, commit)
	}

	return contributions, (*ProofCommit)(commit), nil
}

func (s *ProofStructure) BuildProof(commit *ProofCommit, challenge *big.Int) *Proof {
	result := &Proof{
		Cs:         make([]*big.Int, len(commit.c)),
		DResponses: make([]*big.Int, len(commit.d)),
		VResponses: make([]*big.Int, len(commit.v)),
		V5Response: new(big.Int).Add(new(big.Int).Mul(challenge, commit.v5), commit.v5Randomizer),
		MResponse:  new(big.Int).Add(new(big.Int).Mul(challenge, commit.m), commit.mRandomizer),

		Ld: s.ld,
		A:  s.a,
		K:  new(big.Int).Set(s.k),
	}

	for i := range commit.c {
		result.Cs[i] = new(big.Int).Set(commit.c[i])
	}
	for i := range commit.d {
		result.DResponses[i] = new(big.Int).Add(new(big.Int).Mul(challenge, commit.d[i]), commit.dRandomizers[i])
	}
	for i := range commit.v {
		result.VResponses[i] = new(big.Int).Add(new(big.Int).Mul(challenge, commit.v[i]), commit.vRandomizers[i])
	}

	return result
}

func (s *ProofStructure) VerifyProofStructure(g *QrGroup, p *Proof) bool {
	if len(s.cRep) != len(p.Cs) || len(s.cRep) != len(p.DResponses) || len(s.cRep) != len(p.VResponses) {
		return false
	}

	if p.V5Response == nil || p.MResponse == nil {
		return false
	}

	if uint(p.V5Response.BitLen()) > s.lm+s.ld+2+s.lh+s.lstatzk+1 ||
		uint(p.MResponse.BitLen()) > s.lm+s.lh+s.lstatzk+1 {
		return false
	}

	for i := range s.cRep {
		if p.Cs[i] == nil || p.DResponses[i] == nil || p.VResponses[i] == nil {
			return false
		}

		if p.Cs[i].BitLen() > g.N.BitLen() ||
			uint(p.DResponses[i].BitLen()) > s.ld+s.lh+s.lstatzk+1 ||
			uint(p.VResponses[i].BitLen()) > s.lm+s.lh+s.lstatzk+1 {
			return false
		}
	}

	return true
}

func (s *ProofStructure) CommitmentsFromProof(g *QrGroup, p *Proof, challenge *big.Int) []*big.Int {
	bases := keyproof.NewBaseMerge((*qrGroup)(g), (*proof)(p))

	contributions := []*big.Int{}
	contributions = s.mCorrect.commitmentsFromProof((*qrGroup)(g), contributions, challenge, &bases, (*proof)(p))
	for i := range s.cRep {
		contributions = s.cRep[i].commitmentsFromProof((*qrGroup)(g), contributions, challenge, &bases, (*proof)(p))
	}

	return contributions
}

// Check whether proof makes required statement
func (p *Proof) ProvesStatement(a int, k *big.Int) bool {
	if len(p.Cs) == 3 {
		a *= 4
		k = new(big.Int).Mul(k, big.NewInt(4))
		k.Sub(k, big.NewInt(2))
	}
	return a == p.A && k.Cmp(p.K) == 0
}

// Extract proof structure from proof
func (p *Proof) ExtractStructure(lh, lstatzk, lm uint) (*ProofStructure, error) {
	// Check that all values needed for the structure are present and reasonable
	//
	// ld > lm is never reasonable since that implies a difference greater than 2^(2*lm)
	//  which is bigger than m*a
	// p.K >= 2^lm+sizeof(a) is never reasonable since that makes |m*a| < |k|, making
	//  the proof statement trivial (it either always or never holds)
	if p.K == nil || p.Ld > lm || len(p.Cs) < 3 || len(p.Cs) > 4 ||
		p.K.BitLen() > int(lm+strconv.IntSize) {
		return nil, errors.New("Invalid proof")
	}
	return newWithParams(p.A, p.K, nil, len(p.Cs), p.Ld, lh, lstatzk, lm), nil
}

// ---
// Commit structure keyproof interfaces
// ---
func (c *proofCommit) Secret(name string) *big.Int {
	if name == "m" {
		return c.m
	}
	if name == "v5" {
		return c.v5
	}
	if name[0] == 'v' {
		i, err := strconv.Atoi(name[1:])
		if err != nil || i < 0 || i >= len(c.v) {
			return nil
		}
		return c.v[i]
	}
	if name[0] == 'd' {
		i, err := strconv.Atoi(name[1:])
		if err != nil || i < 0 || i >= len(c.d) {
			return nil
		}
		return c.d[i]
	}
	return nil
}

func (c *proofCommit) Randomizer(name string) *big.Int {
	if name == "m" {
		return c.mRandomizer
	}
	if name == "v5" {
		return c.v5Randomizer
	}
	if name[0] == 'v' {
		i, err := strconv.Atoi(name[1:])
		if err != nil || i < 0 || i >= len(c.vRandomizers) {
			return nil
		}
		return c.vRandomizers[i]
	}
	if name[0] == 'd' {
		i, err := strconv.Atoi(name[1:])
		if err != nil || i < 0 || i >= len(c.dRandomizers) {
			return nil
		}
		return c.dRandomizers[i]
	}
	return nil
}

func (c *proofCommit) Base(name string) *big.Int {
	if name[0] == 'C' {
		i, err := strconv.Atoi(name[1:])
		if err != nil || i < 0 || i >= len(c.c) {
			return nil
		}
		return c.c[i]
	}
	return nil
}

func (c *proofCommit) Exp(ret *big.Int, name string, exp, n *big.Int) bool {
	base := c.Base(name)
	if base == nil {
		return false
	}
	ret.Exp(base, exp, n)
	return true
}

func (c *proofCommit) Names() []string {
	result := make([]string, 0, len(c.c))
	for i := range c.c {
		result = append(result, fmt.Sprintf("C%d", i))
	}

	return result
}

// ---
// Proof structure keyproof interfaces
// ---
func (p *proof) ProofResult(name string) *big.Int {
	if name == "m" {
		return p.MResponse
	}
	if name == "v5" {
		return p.V5Response
	}
	if name[0] == 'v' {
		i, err := strconv.Atoi(name[1:])
		if err != nil || i < 0 || i >= len(p.VResponses) {
			return nil
		}
		return p.VResponses[i]
	}
	if name[0] == 'd' {
		i, err := strconv.Atoi(name[1:])
		if err != nil || i < 0 || i >= len(p.DResponses) {
			return nil
		}
		return p.DResponses[i]
	}
	return nil
}

func (p *proof) Base(name string) *big.Int {
	if name[0] == 'C' {
		i, err := strconv.Atoi(name[1:])
		if err != nil || i < 0 || i >= len(p.Cs) {
			return nil
		}
		return p.Cs[i]
	}
	return nil
}

func (p *proof) Exp(ret *big.Int, name string, exp, n *big.Int) bool {
	base := p.Base(name)
	if base == nil {
		return false
	}
	ret.Exp(base, exp, n)
	return true
}

func (p *proof) Names() []string {
	result := make([]string, 0, len(p.Cs))
	for i := range p.Cs {
		result = append(result, fmt.Sprintf("C%d", i))
	}

	return result
}
