package rangeproof

import (
	"errors"

	"github.com/privacybydesign/gabi/big"
	"github.com/privacybydesign/gabi/internal/common"
)

// SquareSplitter provides a combined interface for all facets describing a method for spliting positive numbers into a sum of squares.
type SquareSplitter interface {
	// Number of bits per square
	Ld() uint
	// Number of squares in result
	SquareCount() int
	// Actual splitting function, on input delta, should return array x such that sum_i x_i^2 = delta and len(x) = SquareCount()
	Split(*big.Int) ([]*big.Int, error)
}

type SquaresTable [][]int64

// Generate lookup table for splitting numbers into 3 squares containing entries up-to and including limit
// takes O(n^3/2)
func GenerateSquaresTable(limit int64) SquaresTable {
	result := make(SquaresTable, limit+1)

	// 3 squares can't produce everything, but this is compensated for
	// so we only need to focus on n for which n == 2 (mod 4), with the
	// tradeoff that limit is 4x as large
	for i := int64(0); i*i <= 4*limit; i++ {
		for j := int64(0); i*i+j*j <= 4*limit; j++ {
			for k := int64(0); i*i+j*j+k*k <= 4*limit; k++ {
				v := i*i + j*j + k*k
				if v%4 != 2 {
					continue
				}
				result[(v-2)/4] = []int64{i, j, k}
			}
		}
	}

	return result
}

func (t *SquaresTable) Split(delta *big.Int) ([]*big.Int, error) {
	t_ := [][]int64(*t)
	v := delta.Int64()
	if !delta.IsInt64() || v < 0 || v >= int64(len(t_)) || v%4 != 2 {
		return nil, errors.New("Value outside of table range")
	}

	v = (v - 2) / 4

	return []*big.Int{big.NewInt(t_[v][0]), big.NewInt(t_[v][1]), big.NewInt(t_[v][2])}, nil
}

func (t *SquaresTable) SquareCount() int {
	return 3
}

func (t *SquaresTable) Ld() uint {
	l := len([][]int64(*t))
	ld := uint(0)
	for l > 0 {
		l /= 4
		ld++
	}
	return ld + 1 // compensate for extra bit due to 3-square correction
}

type FourSquaresSplitter struct{}

func (_ *FourSquaresSplitter) Split(delta *big.Int) ([]*big.Int, error) {
	a, b, c, d := common.SumFourSquares(delta)
	return []*big.Int{a, b, c, d}, nil
}

func (_ *FourSquaresSplitter) SquareCount() int {
	return 4
}

func (_ *FourSquaresSplitter) Ld() uint {
	return 128
}