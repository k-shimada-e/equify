# equify

**testify の書き心地で、型の `Equal` メソッドを尊重する Go の assertion ライブラリ。**

[English](README.md) | 日本語

equify は、[testify](https://github.com/stretchr/testify) の呼び出し方と [go-cmp](https://github.com/google/go-cmp) の比較ルールを組み合わせます。`assert.Equal(t, want, got)` をそのまま使い、型の `Equal` メソッドに従って比較できます。構造体・スライス・マップに含まれる型にも適用されます。

- パッケージ関数、`assert.New(t)` のメソッド、`require` 版に対応。
- カスタムメッセージと `Equalf` などの書式付きメッセージに対応。
- 失敗表示は testify と同じ構成。差分本文には go-cmp の出力を使用。
- フィールドの除外や、空のコレクションの同一視などをオプションで指定可能。
- その他の assertion は testify の実装を利用。

## 導入

Go **1.21 以降**が必要です。最低バージョンは固定した依存先の要求に合わせています。testify v1.11.1 は Go 1.17、go-cmp v0.7.0 は Go 1.21 を要求します。

```sh
go get github.com/k-shimada-e/equify@v0.1.1
```

以下の例のように、ルートパッケージを `assert` として、サブパッケージを `require` として import します。

## まず使ってみる

同じ瞬間を表す `time.Time` でも、内部表現が異なる場合があります。equify は `Time.Equal` に従って比較します。

```go
package example_test

import (
	"testing"
	"time"

	"github.com/k-shimada-e/equify/assert"
	"github.com/k-shimada-e/equify/require"
)

func TestInstant(t *testing.T) {
	want := time.Unix(100, 0).UTC()
	got := want.In(time.FixedZone("offset", 3600))

	assert.Equal(t, want, got)
	assert.Equalf(t, want, got, "event %d", 42)
	assert.New(t).Equal(want, got)

	// 失敗した時点でテストを停止する場合。
	require.Equal(t, want, got)
	require.New(t).Equal(want, got)
}
```

`assert.Equal`／`assert.NotEqual` は `bool` を返し、失敗後もテストを続行します。`require` 版は戻り値を持たず、失敗時に `FailNow` を呼びます。`require` はテスト本体の goroutine から呼んでください。

## 自分の型の Equal を使う

対応する `Equal` メソッドを持つ型は、その結果で一致・不一致を判定します。次の例では `ID` だけを比較します。

```go
package example_test

import (
	"testing"

	"github.com/k-shimada-e/equify/assert"
)

type User struct {
	ID   int
	Name string
}

func (u User) Equal(other User) bool {
	return u.ID == other.ID
}

func TestUsers(t *testing.T) {
	want := User{ID: 1, Name: "Alice"}
	got := User{ID: 1, Name: "Alicia"}

	assert.Equal(t, want, got) // ID が同じなので成功。
	assert.Equal(t, []User{want}, []User{got}) // 要素にも User.Equal を適用。
}
```

メソッドは `Equal(T) bool` や `Equal(interface{}) bool` など、[go-cmp が認識するシグネチャ](https://pkg.go.dev/github.com/google/go-cmp/cmp#Equal)に従う必要があります。任意の名前の比較メソッドを自動で呼ぶわけではありません。`Equal` は結果が決定的かつ対称になるように実装してください。

## 失敗時の表示

上の `User` で異なる ID を比較すると、次のように表示します。パス・行番号・空白などは実行環境によって変わります。

```text
Error Trace: example_test.go:42
Error:       Not equal:
             expected: example_test.User{ID:1, Name:"Alice"}
             actual  : example_test.User{ID:2, Name:"Bob"}

             Diff:
             --- Expected
             +++ Actual
               example_test.User(
             -   {ID: 1, Name: "Alice"},
             +   {ID: 2, Name: "Bob"},
               )
Messages:    user comparison
```

`Messages` は、`assert.Equal(t, want, got, "user comparison")` のようにメッセージを指定した場合に表示します。

`Not equal:`・`expected`／`actual`・`Diff:` の構成と、期待値・実際の値の要約は testify に揃えています。差分は `-` が期待値、`+` が実際の値です。差分本文は go-cmp の形式です。

`Equal` メソッドは `bool` しか返さないため、不一致の理由までは取得できません。その比較箇所の値全体を表示し、メソッドが比較に使わないフィールドも表示対象になります。`Equal` メソッドを持たず、フィールドを再帰的に比較する型では、フィールド単位の差分を表示します。

## 比較をカスタマイズする

既存の `Equal` のシグネチャを維持するため、オプションは `NewWithOptions` または専用関数で指定します。

```go
package example_test

import (
	"testing"

	"github.com/k-shimada-e/equify/assert"
	"github.com/k-shimada-e/equify/require"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

type Record struct {
	ID    int
	Cache string
}

func TestComparisonOptions(t *testing.T) {
	// このインスタンスの等価比較にオプションを適用。
	a := assert.NewWithOptions(t, cmpopts.EquateEmpty())
	a.Equal([]int(nil), []int{})

	// 1 回の比較だけにオプションを指定。
	want := Record{ID: 1, Cache: "old"}
	got := Record{ID: 1, Cache: "new"}
	assert.EqualWithOptions(t, want, got,
		[]cmp.Option{cmpopts.IgnoreFields(Record{}, "Cache")})

	// require 側も同様に設定できます。
	r := require.NewWithOptions(t, cmpopts.EquateEmpty())
	r.Equal([]int(nil), []int{})
}
```

`assert.NotEqualWithOptions` も利用できます。インスタンスのオプションは `Equal`／`Equalf`／`NotEqual`／`NotEqualf` に適用し、`EqualValues` などの委譲先のメソッドには適用しません。

## testify からの移行

import を置き換え、既存の assertion 呼び出しを維持できます。

```diff
- "github.com/stretchr/testify/assert"
- "github.com/stretchr/testify/require"
+ "github.com/k-shimada-e/equify/assert"
+ "github.com/k-shimada-e/equify/require"
```

比較ルールが変わるため、移行後はテストを実行してください。

### 比較方法が変わる API

| API | 動作 |
| --- | --- |
| `Equal`／`Equalf` | go-cmp で比較し、不一致なら差分を表示 |
| `NotEqual`／`NotEqualf` | 同じ比較ルールで、一致したら失敗 |
| `assert.ObjectsAreEqual` | 同じ比較ルールで `bool` を返す |
| その他の公開 API | testify の実装に委譲 |

`Equal`／`NotEqual` 系の変更は、パッケージ関数・assertion メソッド・`require` 版に適用します。

`EqualValues`・`NotEqualValues`・`EqualExportedValues`・`Exactly`・`ElementsMatch`・`ObjectsAreEqualValues` などの内部比較は testify のままです。ライブラリ全体の比較処理を go-cmp に置き換えるものではありません。

### 互換性の範囲

初版は **testify v1.11.1** の公開パッケージ関数のシグネチャと、`New(t)` の assertion メソッドを対象にしています。

- equify の `*Assertions` は upstream の `*Assertions` とは別の型です。upstream の具体的な型を受け取るコードでは変更が必要です。
- `testify/suite` が内部で生成する assertion は testify のままです。equify の関数またはインスタンスを明示的に呼んでください。
- `testify/mock`・`testify/suite` の代替パッケージは提供しません。
- 失敗表示の構成は揃えていますが、差分本文は完全一致しません。

### デフォルトの比較ルール

- 対応する `Equal` メソッドがあれば、入れ子の値も含めてその結果を尊重します。
- メソッドがない構造体は、非公開フィールドも比較します。標準の go-cmp とは異なり、内部で `cmp.Exporter` によるアクセスを許可しています。
- `nil` と空のスライス、および `int` と `int64` は区別します。
- `Equal`／`NotEqual` のトップレベル引数に関数を渡すと、testify と同様に assertion が失敗します。
- 不正な比較オプションや比較メソッドが発生させた panic はそのまま伝播します。

循環参照など、その他の比較ルールは go-cmp に従います。オプションがない場合も、testify の `reflect.DeepEqual` にフォールバックしません。

## 開発

依存バージョンは `go.mod`・`go.sum` で固定しています。

```sh
go mod download
go test ./...
go vet ./...
```

CI は testify と同様に最低対応バージョンと現行の安定版を検証する方針です。Linux・Windows と Go 1.21・一つ前の安定版（`oldstable`）・最新安定版（`stable`）の組み合わせでテストと `go vet` を実行します。別の Linux ジョブで `gofmt`、生成コード、モジュール情報の整合性、race 検出も確認します。ルートのテストから呼び出す `require` を含めて、パッケージ横断のカバレッジを取得するには次を実行してください。

```sh
go test -count=1 -timeout=5m '-coverpkg=./...' '-coverprofile=coverage.out' ./...
go tool cover '-func=coverage.out'
go test -race -count=1 -timeout=5m ./...
```

race 検出には対応する C コンパイラが必要です。カバレッジには生成された委譲関数も含まれるため、達成率のしきい値は設けていません。

`assert/forward.go` と `require/forward.go` は、固定した testify のソースから生成した委譲関数です。手動で編集せず、依存バージョンを変更したら再生成して検証してください。

```sh
go generate ./...
go test ./...
go vet ./...
```

テストでは、独自の `Equal` メソッドと入れ子の値、`time.Time`、比較オプション、testify 形式の失敗時の要約、呼び出し元の行番号、`require` の即時停止を確認しています。

## ライセンス

testify と同じ [MIT License](LICENSE) です。失敗時の要約表示は testify の実装を基にしており、その著作権表記をライセンスファイルに残しています。依存パッケージにはそれぞれのライセンスが適用されます。
