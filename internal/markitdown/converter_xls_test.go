// Copyright 2026 Conductor OSS
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on
// an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the
// specific language governing permissions and limitations under the License.

package markitdown

import (
	"testing"

	"github.com/extrame/xls"
)

// TestSafeRowCellsMissingRowNoPanic 复现 extrame/xls 的 nil 解引用 panic：
// 当工作表的 rows 表中不存在某个行下标（稀疏行或畸形文件）时，
// Row() 内部会解引用空指针导致整个进程崩溃。
// 零值 WorkSheet 的 rows 为 nil，等价于"所有行都缺失"的最坏场景。
// safeRowCells 用 recover 统一兜底 Row() + Col() 全链路。
func TestSafeRowCellsMissingRowNoPanic(t *testing.T) {
	sheet := &xls.WorkSheet{}

	for i := 0; i < 10; i++ {
		if cells := safeRowCells(sheet, i); cells != nil {
			t.Fatalf("safeRowCells(%d) = %v, want nil", i, cells)
		}
	}
}
