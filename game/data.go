package main

import "image/color"

// Rarity of a gacha result.
type Rarity int

const (
	RarityN Rarity = iota
	RarityR
	RaritySR
	RaritySSR
)

func (r Rarity) String() string {
	return [...]string{"N", "R", "SR", "SSR"}[r]
}

var rarityColors = [...]color.RGBA{
	RarityN:   {0xc8, 0xc8, 0xd0, 0xff},
	RarityR:   {0x5c, 0xc8, 0xff, 0xff},
	RaritySR:  {0xff, 0xc8, 0x3c, 0xff},
	RaritySSR: {0xff, 0x5c, 0xd0, 0xff},
}

// rarityWeights are the relative draw weights per rarity.
var rarityWeights = [...]int{RarityN: 50, RarityR: 30, RaritySR: 15, RaritySSR: 5}

type TalentID string

const (
	TDoubleJump TalentID = "double_jump"
	TTough      TalentID = "tough"
	TMagnet     TalentID = "magnet"
	TEarly      TalentID = "early_bird"
	TBugProof   TalentID = "bug_proof"
	TForesight  TalentID = "foresight"
	TLucky      TalentID = "lucky"
	TPushup     TalentID = "pushup"
	TLoud       TalentID = "loud"
	TLost       TalentID = "lost"
	TOversleep  TalentID = "oversleep"
	TDodge      TalentID = "dodge"
	TScholar    TalentID = "scholar"
	TThrifty    TalentID = "thrifty"
	TGambler    TalentID = "gambler"
	TTall       TalentID = "tall"
	TCatLover   TalentID = "cat_lover"
	TSprinter   TalentID = "sprinter"
	TPhoenix    TalentID = "phoenix"
	TDebugger   TalentID = "debugger"
)

type Talent struct {
	ID     TalentID
	Name   string
	Rarity Rarity
	Desc   string
}

var talents = []Talent{
	{TDoubleJump, "2段ジャンプ", RaritySR, "たまに2段のぼれる"},
	{TTough, "強心臓", RaritySR, "ライフ+1"},
	{TMagnet, "磁石体質", RarityR, "となりの実績も拾う"},
	{TEarly, "早起き", RarityR, "時間が10%おそい"},
	{TBugProof, "バグ耐性", RarityR, "バグを1回むこう化"},
	{TForesight, "先読み", RaritySR, "ルートが遠くまで見える"},
	{TLucky, "強運", RaritySSR, "レアイベント2倍"},
	{TPushup, "腕立て伏せが速い", RarityN, "特に効果なし"},
	{TLoud, "声が大きい", RarityN, "ジャンプ音が大きい"},
	{TLost, "方向音痴", RarityN, "たまに左右がいれかわる"},
	{TOversleep, "寝坊", RarityN, "時間がすぐ後ろにいる"},
	{TDodge, "逃げ足", RarityR, "敵が15%おそい"},
	{TScholar, "勉強家", RarityR, "実績のポイント+20%"},
	{TThrifty, "倹約家", RarityN, "人生ポイント+5%"},
	{TGambler, "ギャンブラー", RaritySR, "レアイベントの点が大ブレ"},
	{TTall, "高身長", RarityN, "特に効果なし（目立つ）"},
	{TCatLover, "猫好き", RarityN, "猫がついてくる"},
	{TSprinter, "瞬発力", RarityR, "時間をはなしてスタート"},
	{TPhoenix, "不死鳥", RaritySSR, "時間に追いつかれても1回復活"},
	{TDebugger, "デバッガー", RaritySR, "バグ足場が赤く見える"},
}

func talentByID(id TalentID) (Talent, bool) {
	for _, t := range talents {
		if t.ID == id {
			return t, true
		}
	}
	return Talent{}, false
}

type DifficultyID string

type Difficulty struct {
	ID        DifficultyID
	Name      string
	Weight    int
	Rarity    Rarity
	LifeBonus int
	EnemyMul  float64
	TimeMul   float64
	BugMul    float64
	PointMul  float64
}

var difficulties = []Difficulty{
	{"easy", "EASY", 15, RarityR, 1, 0.6, 0.85, 0.6, 0.8},
	{"normal", "NORMAL", 40, RarityN, 0, 1.0, 1.0, 1.0, 1.0},
	{"hard", "HARD", 30, RarityN, 0, 1.3, 1.15, 1.3, 1.3},
	{"very_hard", "VERY HARD", 12, RaritySR, 0, 1.6, 1.3, 1.6, 1.6},
	{"lunatic", "LUNATIC", 3, RaritySSR, 0, 2.0, 1.45, 3.0, 2.0},
}

type FamilyID string

const (
	FRich    FamilyID = "rich"
	FFree    FamilyID = "free"
	FProtect FamilyID = "protective"
	FMover   FamilyID = "mover"
	FNormal  FamilyID = "normal"
	FBig     FamilyID = "big"
)

type Family struct {
	ID     FamilyID
	Name   string
	Rarity Rarity
	Desc   string
}

var families = []Family{
	{FRich, "実家が太い", RaritySR, "実績1つ持ってスタート"},
	{FFree, "放任主義", RarityR, "時間が速いがポイント+20%"},
	{FProtect, "過保護", RarityR, "20歳まで1回だけ守られる"},
	{FMover, "転勤族", RarityN, "背景がころころ変わる"},
	{FNormal, "ごく普通", RarityN, "普通が一番"},
	{FBig, "大家族", RarityR, "たまに家族がかばってくれる"},
}

func familyByID(id FamilyID) Family {
	for _, f := range families {
		if f.ID == id {
			return f
		}
	}
	return families[4]
}

// Stage is a period of life determined by age.
type Stage struct {
	Name         string
	FromAge      float64
	Sky, Ground  color.RGBA
	Enemies      []string
	Achievements []string
}

var stages = []Stage{
	{"幼少期", 0, color.RGBA{0xff, 0xe0, 0xb8, 0xff}, color.RGBA{0xf0, 0xa8, 0x78, 0xff},
		[]string{"ピーマン", "歯医者", "おばけ", "注射"}, []string{"逆上がり", "ひらがな", "お手伝い"}},
	{"学生", 6, color.RGBA{0xb8, 0xe8, 0xff, 0xff}, color.RGBA{0x68, 0xa8, 0xe0, 0xff},
		[]string{"宿題", "小テスト", "受験", "校則", "反抗期"}, []string{"皆勤賞", "卒業証書", "部活の大会"}},
	{"青春", 18, color.RGBA{0xff, 0xc0, 0xd8, 0xff}, color.RGBA{0xe0, 0x70, 0xa8, 0xff},
		[]string{"単位", "バイトのシフト", "二日酔い", "就活"}, []string{"内定", "運転免許", "初任給"}},
	{"社会人", 22, color.RGBA{0xc8, 0xd0, 0xe0, 0xff}, color.RGBA{0x70, 0x78, 0x98, 0xff},
		[]string{"上司", "締め切り", "請求書", "満員電車", "会議", "残業"}, []string{"昇進", "資格", "マイホーム", "ボーナス"}},
	{"老後", 65, color.RGBA{0xe8, 0xd8, 0xa8, 0xff}, color.RGBA{0xa8, 0x90, 0x58, 0xff},
		[]string{"年金の通知", "腰痛", "物忘れ", "健康診断"}, []string{"孫", "盆栽", "ゲートボール優勝"}},
	{"仙人", 100, color.RGBA{0xe0, 0xf8, 0xe8, 0xff}, color.RGBA{0x88, 0xc8, 0xa0, 0xff},
		[]string{"霞", "雷雲", "修行僧"}, []string{"悟り", "長寿番付"}},
	{"妖怪", 200, color.RGBA{0x48, 0x38, 0x68, 0xff}, color.RGBA{0x28, 0x18, 0x40, 0xff},
		[]string{"お札", "陰陽師", "朝日"}, []string{"妖力", "百鬼夜行"}},
	{"神話", 1000, color.RGBA{0x10, 0x10, 0x30, 0xff}, color.RGBA{0x30, 0x20, 0x60, 0xff},
		[]string{"隕石", "ブラックホール", "ビッグバン"}, []string{"星座入り", "伝説"}},
}

func stageIndexForAge(age float64) int {
	idx := 0
	for i, s := range stages {
		if age >= s.FromAge {
			idx = i
		}
	}
	return idx
}

type RareEvent struct {
	Text string
}

var rareEvents = []RareEvent{
	{"宝くじが当たった！"},
	{"庭から石油が出た！"},
	{"タイムマシンを拾った（こわれてた）"},
	{"なぜかバズった！"},
	{"徳を積んだ"},
	{"UFOに手をふられた"},
	{"四つ葉のクローバーを見つけた"},
}

// Look is the randomly generated appearance.
type Look struct {
	Hair, Hat, Clothes int
}

var (
	hairColors = []color.RGBA{
		{0x30, 0x20, 0x18, 0xff}, {0x80, 0x50, 0x28, 0xff}, {0xf0, 0xd0, 0x60, 0xff},
		{0xe0, 0x48, 0x38, 0xff}, {0x60, 0x80, 0xff, 0xff}, {0xe8, 0xe8, 0xe8, 0xff},
	}
	clothesColors = []color.RGBA{
		{0xe8, 0x40, 0x50, 0xff}, {0x40, 0x90, 0xe8, 0xff}, {0x40, 0xc0, 0x70, 0xff},
		{0xf0, 0xa0, 0x30, 0xff}, {0x90, 0x50, 0xd0, 0xff}, {0x30, 0x30, 0x40, 0xff},
	}
	hatNames = []string{"なし", "キャップ", "リボン", "王冠", "アホ毛"}
)

// Comedic text shown in the result screen depending on age reached.
func epitaph(age float64) string {
	switch {
	case age < 3:
		return "はやすぎる人生"
	case age < 10:
		return "わんぱく盛りでした"
	case age < 20:
		return "青春はこれからだった"
	case age < 40:
		return "働きざかりでした"
	case age < 65:
		return "いぶし銀の人生"
	case age < 100:
		return "大往生（ほぼ）"
	case age < 200:
		return "もはや人間ではない"
	case age < 1000:
		return "妖怪として語りつがれる"
	default:
		return "神話になった"
	}
}
