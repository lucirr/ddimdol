## 구현 계획

### 기술 스택
Go (go.mod 없으므로 새로 초기화)

### 구현 범위
- `_pipeline_test/add.go` — add 함수 구현
- `_pipeline_test/add_test.go` — 테스트

### 브랜치
feature/pipeline-test-add-function

### 테스트 전략
- 정상 케이스: 1+2=3, 0+0=0
- 음수: -1+1=0
- 큰 수: 100+200=300
