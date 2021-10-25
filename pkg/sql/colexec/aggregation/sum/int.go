// Copyright 2021 Matrix Origin
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sum

import (
	"github.com/matrixorigin/matrixone/pkg/container/types"
	"github.com/matrixorigin/matrixone/pkg/container/vector"
	"github.com/matrixorigin/matrixone/pkg/encoding"
	"github.com/matrixorigin/matrixone/pkg/sql/colexec/aggregation"
	"github.com/matrixorigin/matrixone/pkg/vectorize/sum"
	"github.com/matrixorigin/matrixone/pkg/vm/process"
)

func NewInt(typ types.Type) *intSum {
	return &intSum{typ: typ}
}

func (a *intSum) Reset() {
	a.cnt = 0
	a.sum = 0
}

func (a *intSum) Type() types.Type {
	return a.typ
}

func (a *intSum) Dup() aggregation.Aggregation {
	return &intSum{typ: a.typ}
}

func (a *intSum) Fill(sels []int64, vec *vector.Vector) error {
	if n := len(sels); n > 0 {
		switch vec.Typ.Oid {
		case types.T_int8:
			a.sum += sum.Int8SumSels(vec.Col.([]int8), sels)
		case types.T_int16:
			a.sum += sum.Int16SumSels(vec.Col.([]int16), sels)
		case types.T_int32:
			a.sum += sum.Int32SumSels(vec.Col.([]int32), sels)
		case types.T_int64:
			a.sum += sum.Int64SumSels(vec.Col.([]int64), sels)
		}
		a.cnt += int64(n - vec.Nsp.FilterCount(sels))
	} else {
		switch vec.Typ.Oid {
		case types.T_int8:
			a.sum += sum.Int8Sum(vec.Col.([]int8))
		case types.T_int16:
			a.sum += sum.Int16Sum(vec.Col.([]int16))
		case types.T_int32:
			a.sum += sum.Int32Sum(vec.Col.([]int32))
		case types.T_int64:
			a.sum += sum.Int64Sum(vec.Col.([]int64))
		}
		a.cnt += int64(vec.Length() - vec.Nsp.Length())
	}
	return nil
}

func (a *intSum) Eval() interface{} {
	if a.cnt == 0 {
		return nil
	}
	return a.sum
}

func (a *intSum) EvalCopy(proc *process.Process) (*vector.Vector, error) {
	data, err := proc.Alloc(8)
	if err != nil {
		return nil, err
	}
	vec := vector.New(a.typ)
	vs := encoding.DecodeInt64Slice(data[:8])
	vs[0] = a.sum
	if a.cnt == 0 {
		vec.Nsp.Add(0)
	}
	vec.Col = vs
	vec.Data = data
	return vec, nil
}
