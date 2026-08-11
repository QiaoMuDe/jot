package store

import (
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"

	// blank import 注册 sqlite-vec 扩展（通过 sqlite3_auto_extension 自动生效）
	_ "modernc.org/sqlite/vec"

	"gorm.io/gorm"
)

// ProbeVec 探测 sqlite-vec 扩展是否已加载可用：执行 SELECT vec_version()。
// 返回扩展版本字符串；失败时错误信息会指明"扩展未加载"。
func ProbeVec(db *gorm.DB) (string, error) {
	var version string
	if err := db.Raw("SELECT vec_version()").Scan(&version).Error; err != nil {
		return "", fmt.Errorf("sqlite-vec 扩展未加载或不可用: %w", err)
	}
	version = strings.TrimSpace(version)
	if version == "" {
		return "", fmt.Errorf("sqlite-vec 扩展未加载: vec_version() 返回空结果")
	}
	return version, nil
}

// Float32ToBlob 将 float32 向量按小端序打包为 BLOB 字节（sqlite-vec 存储格式）。
func Float32ToBlob(vec []float32) []byte {
	buf := make([]byte, len(vec)*4)
	for i, f := range vec {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(f))
	}
	return buf
}

// BlobToFloat32 将 BLOB 字节解码回 float32 向量；长度不是 4 的倍数时报错。
func BlobToFloat32(data []byte) ([]float32, error) {
	if len(data)%4 != 0 {
		return nil, fmt.Errorf("向量 BLOB 长度 %d 不是 4 的倍数", len(data))
	}
	out := make([]float32, len(data)/4)
	for i := range out {
		out[i] = math.Float32frombits(binary.LittleEndian.Uint32(data[i*4:]))
	}
	return out, nil
}

// CosineSimilarity 计算两个向量的余弦相似度（0 向量时返回 0 做保护）。
func CosineSimilarity(a, b []float32) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	var dot, na, nb float64
	for i := 0; i < len(a) && i < len(b); i++ {
		fa, fb := float64(a[i]), float64(b[i])
		dot += fa * fb
		na += fa * fa
		nb += fb * fb
	}
	if na == 0 || nb == 0 {
		return 0 // 任一为零向量，相似度无意义
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

// VecF32SQL 生成 sqlite-vec 的向量字面量 SQL 片段，如 vec_f32('[0.1,0.2]')。
func VecF32SQL(vec []float32) (string, error) {
	if len(vec) == 0 {
		return "", fmt.Errorf("不能为空向量生成 vec_f32 SQL")
	}
	var sb strings.Builder
	sb.WriteString("vec_f32('[")
	for i, f := range vec {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(strconv.FormatFloat(float64(f), 'g', -1, 32))
	}
	sb.WriteString("]')")
	return sb.String(), nil
}
