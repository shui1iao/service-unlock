// Single retained probe presentation; original MIT notices preserved.
// MIT License; Copyright (c) 2026 Remnawave. See LICENSE.
package render

type purpose int

const (
	video purpose = iota
	music
	aiServices
	gaming
	community
	general
)

var purposeTitles = [...]string{"Video, live & sports", "Music & audio", "AI services", "Gaming", "Community", "General websites"}

type presentation struct {
	purpose purpose
	name    string
}

var corePresentation = map[string]map[string]presentation{
	"geo": {
		"google_captcha": {purpose: general, name: "Google Search (captcha)"},
		"youtube_music":  {purpose: music, name: "YouTube Music"},
		"twitch":         {purpose: video, name: "Twitch (region)"},
		"spotify_signup": {purpose: music, name: "Spotify (Legacy signup endpoint)"},
		"deezer":         {purpose: music, name: "Deezer (region)"},
		"reddit_guest":   {purpose: community, name: "Reddit (guest)"},
		"apple":          {purpose: general, name: "Apple (region)"},
		"steam":          {purpose: gaming, name: "Steam (store region)"},
		"playstation":    {purpose: gaming, name: "PlayStation (region)"},
		"ookla":          {purpose: general, name: "Ookla Speedtest (region)"},
		"jetbrains":      {purpose: general, name: "JetBrains (region)"},
		"bing":           {purpose: general, name: "Microsoft Bing (region)"},
	},
	"access": {
		"chatgpt_web":            {purpose: aiServices, name: "ChatGPT"},
		"youtube_premium_access": {purpose: video, name: "YouTube"},
		"netflix_access":         {purpose: video, name: "Netflix"},
		"claude_access":          {purpose: aiServices, name: "Claude"},
		"tiktok_access":          {purpose: video, name: "TikTok"},
		"gemini_access":          {purpose: aiServices, name: "Gemini"},
	},
	"ai": {
		"deepseek":  {purpose: aiServices, name: "DeepSeek (API)"},
		"qwen_intl": {purpose: aiServices, name: "Qwen Intl (API)"},
		"moonshot":  {purpose: aiServices, name: "Kimi (API)"},
		"zhipu":     {purpose: aiServices, name: "GLM (API)"},
	},
}
var extraPresentation = map[string]presentation{
	"MediaUnlockTest_BahamutAnime":   {purpose: video, name: "Bahamut Anime"},
	"MediaUnlockTest_BilibiliHKMCTW": {purpose: video, name: "BiliBili Hongkong/Macau/Taiwan"},
	"MediaUnlockTest_BilibiliTW":     {purpose: video, name: "Bilibili Taiwan Only"},
	"MediaUnlockTest_AbemaTV":        {purpose: video, name: "Abema.TV"},
	"GameTest_PCRJP":                 {purpose: gaming, name: "Princess Connect Re:Dive Japan"},
	"GameTest_UMAJP":                 {purpose: gaming, name: "Pretty Derby Japan"},
	"GameTest_SDGGGE":                {purpose: gaming, name: "SD Gundam G Generation Eternal"},
	"GameTest_Kancolle":              {purpose: gaming, name: "Kancolle Japan"},
	"MediaUnlockTest_Lemino":         {purpose: video, name: "Lemino"},
	"MediaUnlockTest_AnimeFesta":     {purpose: video, name: "AnimeFesta"},
	"MediaUnlockTest_mora":           {purpose: music, name: "Mora"},
	"MediaUnlockTest_BBCiPLAYER":     {purpose: video, name: "BBC iPLAYER"},
	"MediaUnlockTest_DisneyPlus":     {purpose: video, name: "Disney+"},
	"MediaUnlockTest_Dazn":           {purpose: video, name: "Dazn"},
	"MediaUnlockTest_HuluJP":         {purpose: video, name: "Hulu Japan"},
	"MediaUnlockTest_MyTVSuper":      {purpose: video, name: "MyTVSuper"},
	"MediaUnlockTest_NowE":           {purpose: video, name: "Now E"},
	"MediaUnlockTest_ViuTV":          {purpose: video, name: "Viu.TV"},
	"MediaUnlockTest_unext":          {purpose: video, name: "U-NEXT"},
	"MediaUnlockTest_wowow":          {purpose: video, name: "WOWOW"},
	"MediaUnlockTest_TVer":           {purpose: video, name: "TVer"},
	"MediaUnlockTest_HamiVideo":      {purpose: video, name: "Hami Video"},
	"MediaUnlockTest_4GTV":           {purpose: video, name: "4GTV.TV"},
	"MediaUnlockTest_SlingTV":        {purpose: video, name: "Sling TV"},
	"MediaUnlockTest_PlutoTV":        {purpose: video, name: "Pluto TV"},
	"MediaUnlockTest_HBOMax":         {purpose: video, name: "HBO Max"},
	"MediaUnlockTest_Showmax":        {purpose: video, name: "Showmax"},
	"MediaUnlockTest_Channel4":       {purpose: video, name: "Channel 4"},
	"MediaUnlockTest_ITVHUB":         {purpose: video, name: "ITV Hub"},
	"MediaUnlockTest_DSTV":           {purpose: video, name: "DSTV"},
	"MediaUnlockTest_HuluUS":         {purpose: video, name: "Hulu"},
	"MediaUnlockTest_encoreTVB":      {purpose: video, name: "encoreTVB"},
	"MediaUnlockTest_Molotov":        {purpose: video, name: "Molotov"},
	"MediaUnlockTest_LineTVTW":       {purpose: video, name: "LineTV.TW"},
	"MediaUnlockTest_ViuCom":         {purpose: video, name: "Viu.com"},
	"MediaUnlockTest_Niconico":       {purpose: video, name: "Niconico"},
	"MediaUnlockTest_ParamountPlus":  {purpose: video, name: "Paramount+"},
	"MediaUnlockTest_KKTV":           {purpose: video, name: "KKTV"},
	"MediaUnlockTest_PeacockTV":      {purpose: video, name: "Peacock TV"},
	"MediaUnlockTest_FOD":            {purpose: video, name: "FOD(Fuji TV)"},
	"MediaUnlockTest_BritBox":        {purpose: video, name: "BritBox"},
	"MediaUnlockTest_PrimeVideo":     {purpose: video, name: "Amazon Prime Video"},
	"MediaUnlockTest_Radiko":         {purpose: music, name: "Radiko"},
	"MediaUnlockTest_DMM":            {purpose: general, name: "DMM (financial site)"},
	"MediaUnlockTest_DMMTV":          {purpose: video, name: "DMM TV"},
	"MediaUnlockTest_Catchplay":      {purpose: video, name: "CatchPlay+"},
	"MediaUnlockTest_HotStar":        {purpose: video, name: "HotStar"},
	"MediaUnlockTest_LiTV":           {purpose: video, name: "LiTV"},
	"MediaUnlockTest_FuboTV":         {purpose: video, name: "Fubo TV"},
	"MediaUnlockTest_Fox":            {purpose: video, name: "FOX"},
	"MediaUnlockTest_Joyn":           {purpose: video, name: "Joyn"},
	"MediaUnlockTest_SpotvNow":       {purpose: video, name: "SPOTV NOW"},
	"MediaUnlockTest_SKY_DE":         {purpose: video, name: "SKY DE"},
	"MediaUnlockTest_ZDF":            {purpose: video, name: "ZDF"},
	"MediaUnlockTest_EPIX":           {purpose: video, name: "MGM+"},
	"MediaUnlockTest_NLZIET":         {purpose: video, name: "NLZIET"},
	"MediaUnlockTest_videoland":      {purpose: video, name: "videoland"},
	"MediaUnlockTest_NPO_Start_Plus": {purpose: video, name: "NPO Start Plus"},
	"MediaUnlockTest_RakutenTV":      {purpose: video, name: "Rakuten TV"},
	"MediaUnlockTest_MoviStarPlus":   {purpose: video, name: "Movistar+"},
	"MediaUnlockTest_Starz":          {purpose: video, name: "Starz"},
	"MediaUnlockTest_CanalPlus":      {purpose: video, name: "Canal+"},
	"MediaUnlockTest_Sky_CH":         {purpose: video, name: "SKY CH"},
	"MediaUnlockTest_CBCGem":         {purpose: video, name: "CBC Gem"},
	"MediaUnlockTest_AcornTV":        {purpose: video, name: "Acorn TV"},
	"MediaUnlockTest_Crave":          {purpose: video, name: "Crave"},
	"MediaUnlockTest_Amediateka":     {purpose: video, name: "Amediateka"},
	"MediaUnlockTest_MegogoTV":       {purpose: video, name: "Megogo TV"},
	"MediaUnlockTest_RaiPlay":        {purpose: video, name: "Rai Play"},
	"MediaUnlockTest_TVBAnywhere":    {purpose: video, name: "TVBAnywhere+"},
	"GameTest_ProjectSekai":          {purpose: gaming, name: "Project Sekai: Colorful Stage"},
	"GameTest_KonosubaFD":            {purpose: gaming, name: "Konosuba Fantastic Days"},
	"MediaUnlockTest_NBATV":          {purpose: video, name: "NBA TV"},
	"MediaUnlockTest_Directv":        {purpose: video, name: "Directv Stream"},
	"MediaUnlockTest_SkyGo":          {purpose: video, name: "Sky Go"},
	"MediaUnlockTest_DirecTVGO":      {purpose: video, name: "DirecTV Go"},
	"MediaUnlockTest_DAM":            {purpose: music, name: "Karaoke@DAM"},
	"MediaUnlockTest_DiscoveryPlus":  {purpose: video, name: "Discovery+"},
	"MediaUnlockTest_ESPNPlus":       {purpose: video, name: "ESPN+"},
	"MediaUnlockTest_Binge":          {purpose: video, name: "Binge"},
	"MediaUnlockTest_Docplay":        {purpose: video, name: "Docplay"},
	"MediaUnlockTest_OptusSports":    {purpose: video, name: "Optus Sports"},
	"MediaUnlockTest_KayoSports":     {purpose: video, name: "Kayo Sports"},
	"MediaUnlockTest_NeonTV":         {purpose: video, name: "Neon TV"},
	"MediaUnlockTest_SkyGONZ":        {purpose: video, name: "SkyGo NZ"},
	"MediaUnlockTest_ThreeNow":       {purpose: video, name: "ThreeNow"},
	"MediaUnlockTest_MaoriTV":        {purpose: video, name: "Maori TV"},
	"MediaUnlockTest_SBSonDemand":    {purpose: video, name: "SBS on Demand"},
	"MediaUnlockTest_ABCiView":       {purpose: video, name: "ABC iView"},
	"MediaUnlockTest_Channel9":       {purpose: video, name: "Channel 9"},
	"MediaUnlockTest_Telasa":         {purpose: video, name: "Telasa"},
	"MediaUnlockTest_SetantaSports":  {purpose: video, name: "Setanta Sports"},
	"MediaUnlockTest_Channel5":       {purpose: video, name: "Channel 5"},
	"MediaUnlockTest_MyVideo":        {purpose: video, name: "MyVideo"},
	"MediaUnlockTest_7plus":          {purpose: video, name: "7plus"},
	"MediaUnlockTest_Channel10":      {purpose: video, name: "Channel 10"},
	"MediaUnlockTest_VideoMarket":    {purpose: video, name: "VideoMarket"},
	"MediaUnlockTest_musicjp":        {purpose: music, name: "music.jp"},
	"MediaUnlockTest_Popcornflix":    {purpose: video, name: "Popcornflix"},
	"MediaUnlockTest_TubiTV":         {purpose: video, name: "Tubi TV"},
	"MediaUnlockTest_Philo":          {purpose: video, name: "Philo"},
	"MediaUnlockTest_FXNOW":          {purpose: video, name: "FXNOW"},
	"MediaUnlockTest_Crunchyroll":    {purpose: video, name: "Crunchyroll"},
	"MediaUnlockTest_CWTV":           {purpose: video, name: "CW TV"},
	"MediaUnlockTest_Shudder":        {purpose: video, name: "Shudder"},
	"MediaUnlockTest_TLCGO":          {purpose: video, name: "TLC GO"},
	"MediaUnlockTest_Wavve":          {purpose: video, name: "Wavve"},
	"MediaUnlockTest_Tving":          {purpose: video, name: "Tving"},
	"MediaUnlockTest_CoupangPlay":    {purpose: video, name: "Coupang Play"},
	"MediaUnlockTest_NaverTV":        {purpose: video, name: "Naver TV"},
	"MediaUnlockTest_Afreeca":        {purpose: video, name: "Afreeca TV"},
	"MediaUnlockTest_KBSDomestic":    {purpose: video, name: "KBS Domestic"},
	"MediaUnlockTest_Watcha":         {purpose: video, name: "WATCHA"},
	"MediaUnlockTest_KOCOWA":         {purpose: video, name: "KOCOWA"},
	"MediaUnlockTest_NBCTV":          {purpose: video, name: "NBC TV"},
	"MediaUnlockTest_Crackle":        {purpose: video, name: "Crackle"},
	"MediaUnlockTest_AETV":           {purpose: video, name: "A&E TV"},
	"MediaUnlockTest_NFLPlus":        {purpose: video, name: "NFL+"},
	"MediaUnlockTest_SkyShowTime":    {purpose: video, name: "SkyShowTime"},
	"GameTest_MathsSpot":             {purpose: gaming, name: "MathsSpot Roblox"},
	"MediaUnlockTest_BGlobalSEA":     {purpose: video, name: "B-Global SouthEastAsia"},
	"MediaUnlockTest_BGlobalTH":      {purpose: video, name: "B-Global Thailand Only"},
	"MediaUnlockTest_BGlobalID":      {purpose: video, name: "B-Global Indonesia Only"},
	"MediaUnlockTest_BGlobalVN":      {purpose: video, name: "B-Global Vi\u1ec7t Nam Only"},
	"MediaUnlockTest_AISPlay":        {purpose: video, name: "AIS Play"},
	"WebTest_MetaAI":                 {purpose: aiServices, name: "Meta AI"},
	"WebTest_Wikipedia_Editable":     {purpose: community, name: "Wikipedia Editability"},
	"MediaUnlockTest_MeWatch":        {purpose: video, name: "MeWatch"},
	"MediaUnlockTest_trueID":         {purpose: video, name: "trueID"},
	"MediaUnlockTest_SonyLiv":        {purpose: video, name: "SonyLiv"},
	"WebTest_EroGameSpace":           {purpose: gaming, name: "EroGameSpace"},
	"MediaUnlockTest_DAnimeStore":    {purpose: video, name: "D Anime Store"},
	"MediaUnlockTest_RakutenTVJP":    {purpose: video, name: "Rakuten TV JP"},
	"MediaUnlockTest_AMCPlus":        {purpose: video, name: "AMC+"},
}

func presentationFor(source, id string) presentation {
	if p, ok := corePresentation[source][id]; ok {
		return p
	}
	if source == "extra" {
		if p, ok := extraPresentation[id]; ok {
			return p
		}
	}
	return presentation{purpose: general}
}
func terminalName(source, id, name string) string {
	if p := presentationFor(source, id); p.name != "" {
		return p.name
	}
	if name != "" {
		return source + ":" + id + " (" + name + ")"
	}
	return source + ":" + id
}
