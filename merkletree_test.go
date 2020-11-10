package merkletree

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
)

// drops the unprintable prefix
func bytesToStrForTest(xs []byte) string {
	var xs2 []byte

	for i, c := range xs {
		if c < 128 && c > 31 {
			xs2 = append(xs2, xs[i])
		}
	}

	return string(xs2)
}

func trimNewlines(str string) string {
	return strings.Trim(str, "\n")
}

func expectStrEqual(t *testing.T, actual string, expected string) {
	if trimNewlines(actual) != expected {
		fmt.Println(fmt.Sprintf("=====ACTUAL======\n\n%s\n\n=====EXPECTED======\n\n%s\n", actual, expected))
		t.Fail()
	}
}

var givenOneBlock = trimNewlines(`
(B root: alphaalpha 
  (L root: alpha) 
  (L root: alpha))
`)

var givenFourBlocks = trimNewlines(`
(B root: alphabetakappagamma 
  (B root: alphabeta 
    (L root: alpha) 
    (L root: beta)) 
  (B root: kappagamma 
    (L root: kappa) 
    (L root: gamma)))
`)

var givenTwoBlocks = trimNewlines(`
(B root: alphabeta 
  (L root: alpha) 
  (L root: beta))
`)

var givenThreeBlocks = trimNewlines(`
(B root: alphabetakappakappa 
  (B root: alphabeta 
    (L root: alpha) 
    (L root: beta)) 
  (B root: kappakappa 
    (L root: kappa) 
    (L root: kappa)))
`)

var givenSixBlocks = trimNewlines(`
(B root: alphabetakappagammaepsilonomegaepsilonomega 
  (B root: alphabetakappagamma 
    (B root: alphabeta 
      (L root: alpha) 
      (L root: beta)) 
    (B root: kappagamma 
      (L root: kappa) 
      (L root: gamma))) 
  (B root: epsilonomegaepsilonomega 
    (B root: epsilonomega 
      (L root: epsilon) 
      (L root: omega)) 
    (B root: epsilonomega 
      (L root: epsilon) 
      (L root: omega))))
`)

var proofA = trimNewlines(`
route from omega (leaf) to root:

epsilon + omega = epsilonomega
epsilonomega + muzeta = epsilonomegamuzeta
alphabetakappagamma + epsilonomegamuzeta = alphabetakappagammaepsilonomegamuzeta
`)

func TestCreateMerkleTree(t *testing.T) {
	t.Run("easy tree - just one level (the root) of nodes", func(t *testing.T) {
		blocks := [][]byte{[]byte("alpha"), []byte("beta")}
		tree := NewTree(IdentityHashForTest, blocks)

		expectStrEqual(t, tree.ToString(bytesToStrForTest, 0), givenTwoBlocks)
	})

	t.Run("two levels of nodes", func(t *testing.T) {
		blocks := [][]byte{[]byte("alpha"), []byte("beta"), []byte("kappa"), []byte("gamma")}
		tree := NewTree(IdentityHashForTest, blocks)

		expectStrEqual(t, tree.ToString(bytesToStrForTest, 0), givenFourBlocks)
	})

	t.Run("one block - one level", func(t *testing.T) {
		blocks := [][]byte{[]byte("alpha")}
		tree := NewTree(IdentityHashForTest, blocks)

		expectStrEqual(t, tree.ToString(bytesToStrForTest, 0), givenOneBlock)
	})

	/*

				duplicate a leaf

		            123{3}
				 /        \
			   12          3{3}
			 /    \      /    \
			1      2    3      {3}

	*/
	t.Run("duplicate a leaf to keep the binary tree balanced", func(t *testing.T) {
		blocks := [][]byte{[]byte("alpha"), []byte("beta"), []byte("kappa")}
		tree := NewTree(IdentityHashForTest, blocks)

		expectStrEqual(t, tree.ToString(bytesToStrForTest, 0), givenThreeBlocks)
	})

	/*

			          duplicate a node

		                123456{56}
		          /                    \
		        1234                  56{56}
		     /        \              /      \
		   12          34          56        {56}
		 /    \      /    \      /    \     /    \
		1      2    3      4    5      6  {5}    {6}

	*/
	t.Run("duplicate a branch to keep the tree balanced", func(t *testing.T) {
		blocks := [][]byte{[]byte("alpha"), []byte("beta"), []byte("kappa"), []byte("gamma"), []byte("epsilon"), []byte("omega")}
		tree := NewTree(IdentityHashForTest, blocks)

		expectStrEqual(t, tree.ToString(bytesToStrForTest, 0), givenSixBlocks)
	})
}

func TestAuditProof(t *testing.T) {
	t.Run("Tree#CreateProof", func(t *testing.T) {
		blocks := [][]byte{
			[]byte("alpha"),
			[]byte("beta"),
			[]byte("kappa"),
		}

		tree := NewTree(IdentityHashForTest, blocks)
		target := tree.checksumFunc(true, []byte("alpha"))

		proof, err := tree.CreateProof(target)
		if err != nil {
			t.Fail()
		}

		expected := Proof{
			parts: []*ProofPart{{
				isRight:  true,
				checksum: tree.checksumFunc(true, []byte("beta")),
			}, {
				isRight:  true,
				checksum: tree.checksumFunc(false, append(tree.checksumFunc(true, []byte("kappa")), tree.checksumFunc(true, []byte("kappa"))...)),
			}},
			target: target,
		}

		if !expected.Equals(proof) {
			t.Fail()
		}
	})

	t.Run("Proof#ToString", func(t *testing.T) {
		blocks := [][]byte{
			[]byte("alpha"),
			[]byte("beta"),
			[]byte("kappa"),
			[]byte("gamma"),
			[]byte("epsilon"),
			[]byte("omega"),
			[]byte("mu"),
			[]byte("zeta"),
		}

		tree := NewTree(IdentityHashForTest, blocks)
		target := tree.checksumFunc(true, []byte("omega"))
		proof, _ := tree.CreateProof(target)

		expectStrEqual(t, proof.ToString(bytesToStrForTest), proofA)
	})

	t.Run("Tree#VerifyProof", func(t *testing.T) {
		t.Run("valid proof for a two-leaf tree", func(t *testing.T) {
			blocks := [][]byte{
				[]byte("alpha"),
				[]byte("beta"),
			}

			tree := NewTree(IdentityHashForTest, blocks)

			proof := &Proof{
				parts: []*ProofPart{{
					isRight:  true,
					checksum: tree.checksumFunc(true, []byte("beta")),
				}},
				target: tree.checksumFunc(true, []byte("alpha")),
			}

			if !tree.VerifyProof(proof) {
				t.Fail()
			}
		})

		t.Run("invalid proof (isRight should be true) for a two-leaf tree", func(t *testing.T) {
			blocks := [][]byte{
				[]byte("alpha"),
				[]byte("beta"),
			}

			tree := NewTree(IdentityHashForTest, blocks)

			proof := &Proof{
				parts: []*ProofPart{{
					isRight:  false,
					checksum: tree.checksumFunc(true, []byte("beta")),
				}},
				target: tree.checksumFunc(true, []byte("alpha")),
			}

			if tree.VerifyProof(proof) {
				t.Fail()
			}
		})

		t.Run("invalid proof (wrong sibling) for a two-leaf tree", func(t *testing.T) {
			blocks := [][]byte{
				[]byte("alpha"),
				[]byte("beta"),
			}

			tree := NewTree(IdentityHashForTest, blocks)

			proof := &Proof{
				parts: []*ProofPart{{
					isRight:  true,
					checksum: tree.checksumFunc(true, []byte("kappa")),
				}},
				target: tree.checksumFunc(true, []byte("alpha")),
			}

			if tree.VerifyProof(proof) {
				t.Fail()
			}
		})

		t.Run("invalid proof (tree doesn't contain target) for a two-leaf tree", func(t *testing.T) {
			blocks := [][]byte{
				[]byte("alpha"),
				[]byte("beta"),
			}

			tree := NewTree(IdentityHashForTest, blocks)

			proof := &Proof{
				parts: []*ProofPart{{
					isRight:  true,
					checksum: tree.checksumFunc(true, []byte("beta")),
				}},
				target: tree.checksumFunc(true, []byte("kappa")),
			}

			if tree.VerifyProof(proof) {
				t.Fail()
			}
		})

		t.Run("valid proof for eight leaf tree", func(t *testing.T) {
			blocks := [][]byte{
				[]byte("alpha"),
				[]byte("beta"),
				[]byte("kappa"),
				[]byte("gamma"),
				[]byte("epsilon"),
				[]byte("omega"),
				[]byte("mu"),
				[]byte("zeta"),
			}

			tree := NewTree(IdentityHashForTest, blocks)
			target := tree.checksumFunc(true, []byte("alpha"))

			proof, err := tree.CreateProof(target)
			if err != nil {
				t.Fail()
			}

			if !tree.VerifyProof(proof) {
				t.Fail()
			}
		})
	})
}

func TestHandlesPreimageAttack(t *testing.T) {
	blocks := [][]byte{
		[]byte("alpha"),
		[]byte("beta"),
		[]byte("kappa"),
	}

	tree := NewTree(Sha256DoubleHash, blocks)

	l := append(tree.checksumFunc(true, []byte("alpha")), tree.checksumFunc(true, []byte("beta"))...)
	r := append(tree.checksumFunc(true, []byte("kappa")), tree.checksumFunc(true, []byte("kappa"))...)

	tree2 := NewTree(Sha256DoubleHash, [][]byte{l, r})

	if bytes.Equal(tree.root.GetChecksum(), tree2.root.GetChecksum()) {
		t.Fail()
	}
}

func TestDocsCreateAndPrintAuditProof(t *testing.T) {
	blocks := [][]byte{
		[]byte("alpha"),
		[]byte("beta"),
		[]byte("kappa"),
	}

	tree := NewTree(Sha256DoubleHash, blocks)
	target := tree.checksumFunc(true, []byte("alpha"))
	proof, _ := tree.CreateProof(target)

	fmt.Println(proof.ToString(func(bytes []byte) string {
		return hex.EncodeToString(bytes)[0:16]
	}))
}

func TestDocsCreateAndPrintTree(t *testing.T) {
	blocks := [][]byte{
		[]byte("alpha"),
		[]byte("beta"),
		[]byte("kappa"),
	}

	tree := NewTree(Sha256DoubleHash, blocks)

	fmt.Println(tree.ToString(func(bytes []byte) string {
		return hex.EncodeToString(bytes)[0:16]
	}, 0))
}

func TestDocsValidateProof(t *testing.T) {
	blocks := [][]byte{
		[]byte("alpha"),
		[]byte("beta"),
		[]byte("kappa"),
	}

	tree := NewTree(Sha256DoubleHash, blocks)

	proof, err := tree.CreateProof(tree.rows[0][0].GetChecksum())
	if err != nil {
		panic(err)
	}

	tree.VerifyProof(proof) // true
}
