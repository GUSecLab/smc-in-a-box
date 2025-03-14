package ligero

import (
	"fmt"
)

func AddMatrix(matrix1 [][]int, matrix2 [][]int, q int) [][]int {
	result := make([][]int, len(matrix1))
	for i, a := range matrix1 {
		for j := range a {
			result[i] = append(result[i], mod(matrix1[i][j]+matrix2[i][j], q))
		}
	}
	return result
}

func SubMatrix(matrix1 [][]int, matrix2 [][]int, q int) [][]int {
	result := make([][]int, len(matrix1))
	for i, a := range matrix1 {
		for j := range a {
			result[i] = append(result[i], mod(matrix1[i][j]-matrix2[i][j], q))
		}
	}
	return result
}

func MulMatrix(matrix1, matrix2 [][]int, q int) ([][]int, error) {

	rows1, cols1 := len(matrix1), len(matrix1[0])
	rows2, cols2 := len(matrix2), len(matrix2[0])

	if cols1 != rows2 {
		return nil, fmt.Errorf("Matrix multiplication is not possible. The number of columns in the first matrix must be equal to the number of rows in the second matrix.")

	}

	result := make([][]int, rows1)
	for i := range result {
		result[i] = make([]int, cols2)
	}

	for i := 0; i < rows1; i++ {
		for j := 0; j < cols2; j++ {
			for k := 0; k < cols1; k++ {
				result[i][j] += matrix1[i][k] * matrix2[k][j]
			}
			result[i][j] = mod(result[i][j], q)
		}
	}
	return result, nil
}

func MulList(list1 []int, list2 []int, q int) (int, error) {
	if len(list1) != len(list2) {
		return 0, fmt.Errorf("Invalid inputs: inputs length are different so that multiplication cannot be done")
	}
	result := 0
	for i := 0; i < len(list1); i++ {
		result += list1[i] * list2[i]
	}

	return mod(result, q), nil
}

func (zk *LigeroZK) Interpolate_at_Point(x_samples []int, y_samples []int, x int, q int) (int, error) {
	if len(x_samples) != len(y_samples) {
		return 0, fmt.Errorf("Invalid inputs: x_samples and y_samples length are different")

	}

	for index, item := range x_samples {
		if item == x {
			return y_samples[index], nil
		}
	}

	if !zk.glob_constants.flag_denom {
		zk.glob_constants.values_denom = make([]int, len(x_samples))
		zk.glob_constants.values_denom = GenerateLagrangeConstants(x_samples, x, q)
		zk.glob_constants.flag_denom = true
	}

	if !zk.glob_constants.flag_num[x-1] {
		zk.glob_constants.values_num[x-1] = make([]int, len(x_samples))

		num := 1
		for j := 0; j < len(x_samples); j++ {
			if x == x_samples[j] {
				fmt.Println("ERROR: EQUAL. Numerators goes to zero!!")
			}
			num = mod(num*(x_samples[j]-x), q)
		}

		for i := 0; i < len(x_samples); i++ {

			zk.glob_constants.values_num[x-1][i] = mod(inverse(x_samples[i]-x, q)*num, q)
		}
		// p.glob_constant_num[x-1] = p.lagrange_constants_for_point(x_samples)
		zk.glob_constants.flag_num[x-1] = true

	}

	y := 0
	for i := 0; i < len(y_samples); i++ {
		y = y + mod(y_samples[i]*zk.glob_constants.values_denom[i]*zk.glob_constants.values_num[x-1][i], q)
	}
	return mod(y, q), nil
}

func (zk *LigeroZK) Interpolate_at_Point_Code_Test(x_samples []int, y_samples []int, x int, q int) (int, error) {
	if len(x_samples) != len(y_samples) {
		return 0, fmt.Errorf("Invalid inputs: x_samples and y_samples length are different")

	}

	for index, item := range x_samples {
		if item == x {
			return y_samples[index], nil
		}
	}
	if !zk.glob_constants_code_test.flag_denom {
		zk.glob_constants_code_test.values_denom = make([]int, len(x_samples))
		zk.glob_constants_code_test.values_denom = GenerateLagrangeConstants(x_samples, x, q)
		zk.glob_constants_code_test.flag_denom = true

	}

	if !zk.glob_constants_code_test.flag_num[x-1] {
		zk.glob_constants_code_test.values_num[x-1] = make([]int, len(x_samples))

		num := 1
		for j := 0; j < len(x_samples); j++ {
			if x == x_samples[j] {
				fmt.Println("ERROR: EQUAL. Numerators goes to zero!! IN INTERPOLATE CODE TEST")
			}
			num = mod(num*(x_samples[j]-x), q)
		}

		for i := 0; i < len(x_samples); i++ {

			zk.glob_constants_code_test.values_num[x-1][i] = mod(inverse(x_samples[i]-x, q)*num, q)
		}
		// p.glob_constant_num[x-1] = p.lagrange_constants_for_point(x_samples)
		zk.glob_constants_code_test.flag_num[x-1] = true

	}

	y := 0
	for i := 0; i < len(y_samples); i++ {
		y = y + mod(y_samples[i]*zk.glob_constants_code_test.values_denom[i]*zk.glob_constants_code_test.values_num[x-1][i], q)
	}
	return mod(y, q), nil
}

// lagrange_constants_for_point returns lagrange constants for the given x
func GenerateLagrangeConstants(x_samples []int, x int, q int) []int {

	constants := make([]int, len(x_samples))
	for i := range constants {
		constants[i] = 0
	}

	for i := 0; i < len(constants); i++ {
		xi := x_samples[i]
		denum := 1
		for j := 0; j < len(constants); j++ {
			if j != i {
				xj := x_samples[j]

				denum = mod(denum*(xj-xi), q)
			}
		}
		constants[i] = mod(inverse(denum, q), q)
	}

	return constants
}

// from http://www.ucl.ac.uk/~ucahcjm/combopt/ext_gcd_python_programs.pdf
func egcd_binary(a int, b int) int {
	u, v, s, t, r := 1, 0, 0, 1, 0
	for (mod(a, 2) == 0) && (mod(b, 2) == 0) {
		a, b, r = a/2, b/2, r+1
	}

	alpha, beta := a, b

	for mod(a, 2) == 0 {
		a = a / 2
		if (mod(u, 2) == 0) && (mod(v, 2) == 0) {
			u, v = u/2, v/2
		} else {
			u, v = (u+beta)/2, (v-alpha)/2
		}

	}

	for a != b {
		if mod(b, 2) == 0 {
			b = b / 2
			if (mod(s, 2) == 0) && (mod(t, 2) == 0) {
				s, t = s/2, t/2
			} else {
				s, t = (s+beta)/2, (t-alpha)/2
			}
		} else if b < a {
			a, b, u, v, s, t = b, a, s, t, u, v
		} else {

			b, s, t = b-a, s-u, t-v
		}

	}

	return s
}

// inverse calculates the inverse of a number
func inverse(a int, q int) int {

	a = (a + q) % q
	b := egcd_binary(a, q)
	return b
}

// mod computes a%b and a could be negative number
func mod(a, b int) int {
	return (a%b + b) % b
}
