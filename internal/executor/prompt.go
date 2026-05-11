package executor

import (
	"fmt"

	"github.com/thuanho/pr-reviewers/internal/store"
)

func BuildReviewPrompt(pr *store.PullRequest, extraRules *string) string {
	prompt := fmt.Sprintf(`Bạn là reviewer cho Pull Request #%s của repository %s. Repository đã được checkout sẵn trong thư mục hiện tại. Hãy thực hiện review các thay đổi của PR.

Sử dụng github_mcp để xem nội dung pull request, các thay đổi trong pull request. Phân tích code về:
- Lỗi logic, bug tiềm ẩn
- Vấn đề bảo mật (SQL injection, XSS, thiếu validation, lộ secret)
- Vấn đề hiệu năng (N+1 query, vòng lặp không cần thiết, thiếu cache)
- Code khó đọc, đặt tên khó hiểu, thiếu nhất quán

QUAN TRỌNG:
- Viết ngắn gọn, đủ ý, không lan man
- Chỉ comment khi thực sự có vấn đề, không khen code tốt
- Tất cả nội dung review PHẢI viết bằng tiếng Việt
- Mỗi comment cần chỉ rõ file, dòng code liên quan

Trả về KẾT QUẢ DUY NHẤT dưới dạng JSON object với cấu trúc:

{
  "summary": "Tóm tắt tổng quan về PR, những điểm chính cần lưu ý (tiếng Việt)",
  "overall_verdict": "approve|comment|request_changes",
  "comments": [
    {
      "file_path": "đường/dẫn/file.go",
      "line_start": 10,
      "line_end": 15,
      "body": "Nội dung góp ý bằng tiếng Việt"
    }
  ]
}

Trong đó:
- summary: Tóm tắt ngắn gọn (2-4 câu) đánh giá tổng quan PR
- overall_verdict: "approve" nếu PR tốt, "comment" nếu có góp ý nhỏ, "request_changes" nếu cần sửa
- comments: Danh sách các góp ý cụ thể, mỗi comment có file_path, line_start, line_end, body
- line_start và line_end là số dòng trong file (bắt đầu từ 1)

CHỈ trả về JSON, không thêm text hay markdown bên ngoài.`, pr.URL, pr.RepoFullName)

	if extraRules != nil && *extraRules != "" {
		prompt += fmt.Sprintf("\n\nQuy tắc bổ sung: %s", *extraRules)
	}

	return prompt
}
