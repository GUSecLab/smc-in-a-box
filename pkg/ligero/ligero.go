package ligero

import (
	crypto_rand "crypto/rand"
	"crypto/sha256"
	"fmt"
	"log"
	"math"
	"math/big"
	"sync"

	"strings"

	"example.com/SMC/pkg/packed"
	"example.com/SMC/pkg/rss"
	merkletree "github.com/wealdtech/go-merkletree"
	"gonum.org/v1/gonum/stat/combin"
)

// n_claim: number of claims
// n_server: number of servers
// m: rows number of rearranged input vecotr in the m*l matrix
// l: columns number of rearranged input vector in the m*l matrix, where n_i = m*l
// t: the maximum number of shares that may be seen without learning anything about the secret;
// use in the secret sharing of each input value
// q: a modulus
// n_encode:the number of shares that each row of rearranged input vector is split into
// n_open_col: number of opened columns

type LigeroZK struct {
	npss                     *packed.PackedSecretSharing
	glob_constants           GlobConstants
	glob_constants_code_test GlobConstantsCodeTest
	n_secret                 int
	n_shares                 int
	m                        int
	l                        int
	n_server                 int
	t                        int
	q                        int
	n_encode                 int
	n_open_col               int
}

type Claim struct {
	Shares []int
	Secret int
}

func NewLigeroZK(N_secret, M, N_server, T, Q, N_open int) (*LigeroZK, error) {
	// m has to larger than 0
	if M <= 0 {
		return nil, fmt.Errorf("m cannot be less than 1")
	}

	if M > N_secret {
		return nil, fmt.Errorf("m cannot be larger than n_secrets")
	}

	if 3*T+1 > N_server {
		return nil, fmt.Errorf("n_server cannot be less than 3t+1")
	}

	if N_open <= 0 {
		return nil, fmt.Errorf("n_open cannot be less than 1")
	}

	//compute total number of shares a secret splits to
	N_shares := combin.Binomial(N_server, T)

	// Calculate l as the upper ceiling of len(slice) divided by m
	L := int(math.Ceil(float64(N_secret) / float64(M)))

	N_encode := 6*N_open + 6*L + 1

	pss, err := packed.NewPackedSecretSharing(N_encode, N_open, L, Q)
	if err != nil {
		log.Fatal(err)
	}

	gc := GlobConstants{flag_num: make([]bool, Q), values_num: make([][]int, Q), flag_denom: false, values_denom: make([]int, Q)}
	gc_codetest := GlobConstantsCodeTest{flag_num: make([]bool, Q), values_num: make([][]int, Q), flag_denom: false, values_denom: make([]int, Q)}

	return &LigeroZK{n_secret: N_secret, n_shares: N_shares, m: M, l: L, n_server: N_server, t: T, q: Q, n_encode: N_encode, n_open_col: N_open, npss: pss, glob_constants: gc, glob_constants_code_test: gc_codetest}, nil
}

func (zk *LigeroZK) GenerateProof(secrets []int) ([]*Proof, error) {
	claims, party_sh, err := zk.preprocess(secrets)
	if err != nil {
		log.Fatal(err)
	}

	extended_witness, err := zk.prepare_extended_witness(claims)
	if err != nil {
		log.Fatal(err)
	}

	seed0 := generate_seeds(zk.n_shares+1, zk.q)
	encoded_witness, err := zk.encode_extended_witness(extended_witness, seed0)
	if err != nil {
		log.Fatal(err)
	}

	encoded_witeness_columnwise, err := ConvertToColumnwise(encoded_witness)
	if err != nil {
		log.Fatal(err)
	}

	//commit to the Extended Witness via Merkle Tree
	tree, leaves, nonces, err := zk.generate_merkletree(encoded_witeness_columnwise)
	if err != nil {
		log.Fatal(err)
	}
	root := tree.Root()

	//generate a vector of random numbers using the hash of merkle tree root as seed
	len1 := zk.m * (1 + zk.n_shares)
	len2 := zk.m
	len3 := zk.m
	h1 := zk.generate_hash([][]byte{root})
	random_vector := RandVector(h1, len1+len2+len3, zk.q)

	//generate code test
	seed1 := generate_seeds(zk.l, zk.q)
	code_mask := zk.generate_mask(seed1)
	r1 := random_vector[:len1]

	q_code, err := zk.generate_code_proof(encoded_witness, r1, code_mask)
	if err != nil {
		log.Fatal(err)
	}

	//generate quadratic test
	seed2 := make([]int, zk.l)
	quadra_mask := zk.generate_mask(seed2)
	r2 := random_vector[len1 : len1+len2]

	q_quadra, err := zk.generate_quadratic_proof(encoded_witness, r2, quadra_mask)
	if err != nil {
		log.Fatal(err)
	}

	//generate linear test
	seed3 := make([]int, zk.l)
	linear_mask := zk.generate_mask(seed3)
	r3 := random_vector[len1+len2:]

	q_linear, err := zk.generate_linear_proof(encoded_witness, r3, linear_mask)
	if err != nil {
		log.Fatal(err)
	}

	//generate FST root
	fst_tree, fst_leaves, err := zk.generate_fst_merkletree(party_sh, seed0)
	if err != nil {
		log.Fatal(err)
	}
	fst_root := fst_tree.Root()

	h2 := zk.generate_hash([][]byte{h1, fst_root, ConvertToByteArray(q_code), ConvertToByteArray(q_quadra), ConvertToByteArray(q_linear)})

	//generate column check
	r4 := RandVector(h2, zk.n_open_col, len(leaves))
	column_check, err := zk.generate_column_check(tree, leaves, r4, nonces, code_mask, quadra_mask, linear_mask, encoded_witeness_columnwise)
	if err != nil {
		log.Fatal(err)
	}

	//generate proof for each party
	proofs := make([]*Proof, zk.n_server)
	for i := 0; i < zk.n_server; i++ {

		fst_proof, err := fst_tree.GenerateProof(fst_leaves[i])
		if err != nil {
			log.Fatal("could not generate fst authentication path")
		}

		proofs[i] = newProof(root, column_check, q_code, q_quadra, q_linear, party_sh[i], seed0, fst_root, fst_proof.Hashes)
	}

	return proofs, nil

}

func (zk *LigeroZK) preprocess(secrets []int) ([]Claim, []Shares, error) {
	n_secret := len(secrets)
	if n_secret == 0 || n_secret != zk.n_secret {
		return nil, nil, fmt.Errorf("Invalid input when generating proof: wrong number of secrets")
	}

	nrss, err := rss.NewReplicatedSecretSharing(zk.n_server, zk.t, zk.q)
	if err != nil {
		log.Fatalf("err: %v", err)
	}

	claims := make([]Claim, n_secret)
	party_sh := make([]Shares, zk.n_server)

	for i := 0; i < n_secret; i++ {
		share_list, party, err := nrss.Split(secrets[i])
		if err != nil {
			log.Fatalf("err: %v", err)
		}

		claims[i] = Claim{Secret: secrets[i], Shares: share_list}
		for j := 0; j < zk.n_server; j++ {
			if i == 0 {
				party_sh[j] = Shares{PartyIndex: j, Index: make([]int, len(party[0])), Values: make([][]int, n_secret)}

			}
			party_sh[j].Values[i] = make([]int, len(party[0]))
			for k, share := range party[j] {
				party_sh[j].Index[k] = share.Index
				party_sh[j].Values[i][k] = share.Value
			}
		}

	}

	return claims, party_sh, nil
}

// Generate shares of each value in the input vector, store them with input values in a matrix, which is called extended witness
// parameter input: client's input vector
func (zk *LigeroZK) prepare_extended_witness(claims []Claim) ([][]int, error) {
	if len(claims) == 0 {
		return nil, fmt.Errorf("Invalid claims: claims are empty")
	}

	if len(claims[0].Shares) != zk.n_shares {
		return nil, fmt.Errorf("Invalid input: Number of shares of each claim is not correct")
	}

	if zk.m > len(claims) {
		return nil, fmt.Errorf("Invalid input: Number of claims must equal or larger than m")
	}

	secrets_num := 1
	rows := zk.m * (secrets_num + zk.n_shares)
	matrix := make([][]int, rows)
	for i := range matrix {
		matrix[i] = make([]int, zk.l)
	}

	index := 0
	for i := 0; i < rows; i = i + secrets_num + zk.n_shares {
		for j := 0; j < zk.l; j++ {

			k := 0
			for k < secrets_num {
				matrix[i+k][j] = claims[index].Secret
				k++
			}
			h := 0
			for h < zk.n_shares {
				matrix[i+k+h][j] = claims[index].Shares[h]
				h++
			}
			index++
		}
	}

	return matrix, nil

}

func (zk *LigeroZK) encode_extended_witness(input [][]int, key []int) ([][]int, error) {
	if len(input) == 0 {
		return nil, fmt.Errorf("Invalid input: Input is empty")
	}

	if len(input) != zk.m*(1+zk.n_shares) || len(input[0]) != zk.l {
		return nil, fmt.Errorf("Invalid input")
	}
	matrix := make([][]int, len(input))
	crs1 := NewCryptoRandSource()

	rand_values := make([]int, len(input))
	for i := 0; i < len(input); i++ {
		nonce := i / (1 + zk.n_shares)
		crs1.Seed(key[i%(1+zk.n_shares)], nonce)
		rand_values[i] = int(crs1.Int63(int64(zk.q)))
		matrix[i] = make([]int, zk.n_encode)
	}

	zk.npss.Split(input[0], rand_values[0])

	// Create channels for concurrent processing
	resultChan := make(chan struct {
		row   []int
		index int
	}, len(input))
	errChan := make(chan error, len(input))

	var wg sync.WaitGroup
	wg.Add(len(input))

	// Process each row concurrently
	for i := 0; i < len(input); i++ {
		go func(i int) {
			defer wg.Done()

			shares, err := zk.npss.Split(input[i], rand_values[i])
			if err != nil {
				errChan <- err
				return
			}
			values := make([]int, zk.n_encode)
			for j := 0; j < zk.n_encode; j++ {
				values[j] = shares[j].Value
			}

			resultChan <- struct {
				row   []int
				index int
			}{index: i, row: values}
		}(i)
	}

	// Close result channel after all goroutines finish
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results from goroutines and maintain order
	for result := range resultChan {
		matrix[result.index] = result.row
	}

	select {
	case err := <-errChan:
		return nil, err
	default:
	}

	return matrix, nil
}

func (zk *LigeroZK) generate_merkletree(input [][]int) (*merkletree.MerkleTree, [][]byte, []int, error) {
	length := len(input)
	if length == 0 {
		return nil, nil, nil, fmt.Errorf("Invalid input: Input is empty")
	}

	// generate a list of nonces
	nonces := generate_seeds(length, zk.q)

	// Create channels for concurrent hashing
	hashedColumns := make(chan struct {
		leaf  []byte
		index int
	}, length)
	errChan := make(chan error, length)

	// Hash each column concurrently
	for i := 0; i < length; i++ {
		go func(i int) {
			list := make([]int, len(input[0])+1)
			list = append(list, input[i]...)
			list = append(list, nonces[i])
			concatenated, err := ConvertColumnToString(list)
			if err != nil {
				errChan <- err
				return
			}
			hashedColumns <- struct {
				leaf  []byte
				index int
			}{index: i, leaf: []byte(concatenated)}
		}(i)
	}

	// Collect results from goroutines and maintain order
	leaves := make([][]byte, length)
	for j := 0; j < length; j++ {
		select {
		case result := <-hashedColumns:
			leaves[result.index] = result.leaf
		case err := <-errChan:
			return nil, nil, nil, err
		}
	}

	// Create a new Merkle Tree from hashed columns
	tree, err := merkletree.New(leaves)
	if err != nil {
		return nil, nil, nil, err
	}

	return tree, leaves, nonces, nil
}

func (zk *LigeroZK) generate_fst_merkletree(party_sh []Shares, seeds []int) (*merkletree.MerkleTree, [][]byte, error) {
	// generate and hash each party's shares
	l1 := len(party_sh)
	l2 := len(party_sh[0].Values)
	l3 := len(party_sh[0].Index)

	if l1 == 0 || l2 == 0 || l3 == 0 {
		log.Fatal("party_sh is invalid")
	}
	leaves := make([][]byte, l1)
	for i := 0; i < l1; i++ {
		list := make([]string, l2*l3+l3)
		index := 0
		for j := 0; j < l2; j++ {
			for n := 0; n < l3; n++ {
				list[index] = fmt.Sprintf("%064b", party_sh[i].Values[j][n])
				index++
			}
		}

		for m := 0; m < l3; m++ {
			list[index] = fmt.Sprintf("%064b", seeds[party_sh[i].Index[m]])
			index++
		}
		concat := strings.Join(list, "")
		leaves[i] = []byte(concat)
	}

	//Create a new Merkle Tree
	tree, err := merkletree.New(leaves)
	if err != nil {
		return nil, nil, err
	}

	return tree, leaves, nil

}

func (zk *LigeroZK) generate_column_check(tree *merkletree.MerkleTree, leaves [][]byte, cols []int, m_nonce []int, c_mask []int, q_mask []int, l_mask []int, input [][]int) ([]OpenedColumn, error) {
	column_check := make([]OpenedColumn, len(cols)) // Adjusted length here

	// Create channels for concurrent processing
	resultChan := make(chan struct {
		col   OpenedColumn
		index int
	}, len(cols))
	errChan := make(chan error, len(cols))

	// Use a wait group to wait for all goroutines to finish
	var wg sync.WaitGroup
	wg.Add(len(cols))

	// Process each column concurrently
	for i := range cols {
		go func(i int) {
			defer wg.Done()
			index := cols[i]
			proof, err := tree.GenerateProof(leaves[index])
			if err != nil {
				errChan <- err
				return
			}
			openedCol := OpenedColumn{
				List:         input[index],
				Index:        index,
				Merkle_nonce: m_nonce[index],
				Code_mask:    c_mask[index],
				Quadra_mask:  q_mask[index],
				Linear_mask:  l_mask[index],
				Authpath:     proof.Hashes,
			}
			resultChan <- struct {
				col   OpenedColumn
				index int
			}{openedCol, i}
		}(i)
	}

	// Close result channel after all goroutines finish
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results from goroutines and maintain order
	for result := range resultChan {
		column_check[result.index] = result.col
	}

	// Check for errors
	select {
	case err := <-errChan:
		return nil, err
	default:
	}

	return column_check, nil
}

// generate proof that is used to check if encoded extended witness is encoded correctly
func (zk *LigeroZK) generate_code_proof(input [][]int, randomness []int, mask []int) ([]int, error) {
	if len(input) == 0 {
		return nil, fmt.Errorf("Invalid input: Input is empty")
	}

	if len(input) != zk.m*(1+zk.n_shares) || len(input[0]) != zk.n_encode {
		return nil, fmt.Errorf("Invalid input")
	}

	//compute q_code
	r_matrix := make([][]int, 1)
	r_matrix[0] = randomness
	mask_matrix := make([][]int, 1)
	mask_matrix[0] = mask

	temp_matrix, err := MulMatrix(r_matrix, input, zk.q)
	if err != nil {
		return nil, err
	}
	q_code := AddMatrix(temp_matrix, mask_matrix, zk.q)
	if len(q_code) != 1 {
		return nil, fmt.Errorf("Invalid q_code")
	}

	proof := make([]int, zk.n_open_col+zk.l)
	for i := 0; i < zk.n_open_col+zk.l; i++ {
		proof[i] = q_code[0][i]
	}

	return proof, nil

}

// generate proof that is used to check if input is a vector of 0/1
func (zk *LigeroZK) generate_quadratic_proof(input [][]int, randomness []int, mask []int) ([]int, error) {
	//fmt.Printf("input:%v\n", input)
	if len(input) == 0 {
		return nil, fmt.Errorf("Invalid input: Input is empty")
	}

	if len(input) != zk.m*(1+zk.n_shares) || len(input[0]) != zk.n_encode {
		return nil, fmt.Errorf("Invalid input")
	}

	//generate q_quadra
	result := make([]int, zk.n_encode)

	index := 0
	for row := 0; row < len(input); row = row + zk.n_shares + 1 {
		for col := 0; col < len(input[0]); col++ {
			result[col] += randomness[index] * input[row][col] * (1 - input[row][col])
			//result[col] = mod(result[col], zk.q)
		}
		index += 1
	}

	//fmt.Printf("input:%v\n", result)
	for i := 0; i < len(result); i++ {
		result[i] = result[i] + mask[i]
		result[i] = mod(result[i], zk.q)
	}

	return result, nil

}

func (zk *LigeroZK) generate_linear_proof(input [][]int, randomness []int, mask []int) ([]int, error) {
	if len(input) == 0 {
		return nil, fmt.Errorf("Invalid input: Input is empty")
	}

	if len(input) != zk.m*(1+zk.n_shares) || len(input[0]) != zk.n_encode {
		return nil, fmt.Errorf("Invalid input")
	}

	//generate q_linear
	result := make([]int, zk.n_encode)

	index := 0
	for row := 0; row < len(input); row = row + zk.n_shares + 1 {
		for col := 0; col < len(input[0]); col++ {
			//result[col] = result[col]+input[row][col]
			temp := input[row][col]
			for j := 1; j < zk.n_shares+1; j++ {
				temp = temp - input[row+j][col]
			}
			result[col] = result[col] + temp*randomness[index]
		}
		index += 1
	}

	for i := 0; i < len(result); i++ {
		result[i] = mod(result[i]+mask[i], zk.q)
	}

	return result, nil

}

func (zk *LigeroZK) generate_mask(seeds []int) []int {

	mask := make([]int, zk.n_encode)

	shares, err := zk.npss.Split(seeds, 1)
	if err != nil {
		log.Fatal(err)
	}

	for j := 0; j < zk.n_encode; j++ {
		mask[j] = shares[j].Value
	}

	return mask
}

func (zk *LigeroZK) generate_hash(input [][]byte) []byte {
	if len(input) == 0 {
		log.Fatal("input of hash function could not be empty")
	}
	size := 0
	for _, d := range input {
		size += len(d)
	}

	concat, i := make([]byte, size), 0
	for _, d := range input {
		i += copy(concat[i:], d)
	}

	hash := sha256.Sum256(concat)

	return hash[:]
}

func generate_seeds(size int, q int) []int {
	seeds := make([]int, size)
	//rand.Seed(time.Now().UnixNano())
	checkMap := map[int]bool{}
	for i := 0; i < size; i++ {
		for {
			value, err := crypto_rand.Int(crypto_rand.Reader, big.NewInt(int64(q)))
			if err == nil && !checkMap[int(value.Int64())] {
				checkMap[int(value.Int64())] = true
				seeds[i] = int(value.Int64())
				break
			}

		}
	}

	return seeds
}
