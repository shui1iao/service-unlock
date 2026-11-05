// Terminal-only explicit platform identities; no measurement/schema changes.
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

var purposeTitles = [...]string{
	"Video, live & sports", "Music & audio", "AI services",
	"Gaming", "Community", "General websites",
}

type presentation struct {
	purpose                    purpose
	platform, name, capability string
	regionLabel                string
	regionExpected             bool // This source actually attempts a country observation.
	regionFirst                bool // Catalogue/Access before Fast/Geo; never fills another source.
}

// Explicit source-qualified IDs: company/name similarity never defines identity.
// YouTube Music, SkyShowTime, DMM TV and the distinct TVB/Viu services stay separate.
var corePresentation = map[string]map[string]presentation{
	"geo": {
		"google":         {purpose: general, platform: "google", name: "Google", regionLabel: "Geo"},
		"google_captcha": {purpose: general, platform: "google", name: "Google", capability: "Search"},
		"youtube":        {purpose: video, platform: "youtube", name: "YouTube", regionLabel: "Geo"},
		"youtube_music":  {purpose: music, platform: "youtube_music", name: "YouTube Music"},
		"twitch":         {purpose: video, platform: "twitch", name: "Twitch", regionLabel: "Geo"},
		"chatgpt":        {purpose: aiServices, platform: "chatgpt", name: "ChatGPT", regionLabel: "Geo"},
		"netflix":        {purpose: video, platform: "netflix", name: "Netflix", regionLabel: "Fast"},
		"spotify":        {purpose: music, platform: "spotify", name: "Spotify", regionLabel: "Geo"},
		"spotify_signup": {purpose: music, platform: "spotify", name: "Spotify", capability: "Legacy signup"},
		"deezer":         {purpose: music, platform: "deezer", name: "Deezer", regionLabel: "Geo"},
		"reddit":         {purpose: community, platform: "reddit", name: "Reddit", regionLabel: "Geo"},
		"reddit_guest":   {purpose: community, platform: "reddit", name: "Reddit", capability: "Guest"},
		"prime":          {purpose: video, platform: "prime", name: "Amazon Prime Video", regionLabel: "Geo"},
		"apple":          {purpose: general, platform: "apple", name: "Apple", regionLabel: "Geo"},
		"steam":          {purpose: gaming, platform: "steam", name: "Steam", capability: "Store region", regionLabel: "Geo"},
		"playstation":    {purpose: gaming, platform: "playstation", name: "PlayStation", regionLabel: "Geo"},
		"tiktok":         {purpose: video, platform: "tiktok", name: "TikTok", regionLabel: "Geo"},
		"ookla":          {purpose: general, platform: "ookla", name: "Ookla Speedtest", regionLabel: "Geo"},
		"jetbrains":      {purpose: general, platform: "jetbrains", name: "JetBrains", regionLabel: "Geo"},
		"bing":           {purpose: general, platform: "bing", name: "Microsoft (Bing)", regionLabel: "Geo"},
	},
	"access": {
		"chatgpt_web":            {purpose: aiServices, platform: "chatgpt", name: "ChatGPT", capability: "Web"},
		"youtube_premium_access": {purpose: video, platform: "youtube", name: "YouTube", capability: "Premium"},
		"netflix_access":         {purpose: video, platform: "netflix", name: "Netflix", capability: "Catalogue", regionLabel: "Catalogue", regionExpected: true, regionFirst: true},
		"claude_access":          {purpose: aiServices, platform: "claude", name: "Claude", capability: "Web", regionLabel: "Geo", regionExpected: true},
		"tiktok_access":          {purpose: video, platform: "tiktok", name: "TikTok", capability: "Web"},
		"gemini_access":          {purpose: aiServices, platform: "gemini", name: "Gemini", capability: "Web", regionLabel: "Geo", regionExpected: true},
	},
	"ai": {
		"openai":    {purpose: aiServices, platform: "chatgpt", name: "ChatGPT", capability: "API"},
		"anthropic": {purpose: aiServices, platform: "claude", name: "Claude", capability: "API"},
		"gemini":    {purpose: aiServices, platform: "gemini", name: "Gemini", capability: "API"},
		"deepseek":  {purpose: aiServices, platform: "deepseek", name: "DeepSeek", capability: "API"},
		"qwen_intl": {purpose: aiServices, platform: "qwen", name: "Qwen", capability: "API SG"},
		"qwen_cn":   {purpose: aiServices, platform: "qwen", name: "Qwen", capability: "API CN"},
		"moonshot":  {purpose: aiServices, platform: "moonshot", name: "Kimi / Moonshot", capability: "API"},
		"zhipu":     {purpose: aiServices, platform: "zhipu", name: "Zhipu GLM", capability: "API"},
	},
}

// All134 current extra IDs are explicit; unknown callers do not inherit a prefix rule.
var extraPresentation = map[string]presentation{
	"MediaUnlockTest_BahamutAnime":   {purpose: video, platform: "extra:MediaUnlockTest_BahamutAnime", name: "Bahamut Anime"},
	"MediaUnlockTest_BilibiliHKMCTW": {purpose: video, platform: "bilibili", name: "Bilibili", capability: "HK-MO-TW", regionLabel: "HK-MO-TW"},
	"MediaUnlockTest_BilibiliTW":     {purpose: video, platform: "bilibili", name: "Bilibili", capability: "TW", regionLabel: "TW"},
	"MediaUnlockTest_AbemaTV":        {purpose: video, platform: "extra:MediaUnlockTest_AbemaTV", name: "Abema.TV"},
	"GameTest_PCRJP":                 {purpose: gaming, platform: "extra:GameTest_PCRJP", name: "Princess Connect Re:Dive Japan"},
	"GameTest_UMAJP":                 {purpose: gaming, platform: "extra:GameTest_UMAJP", name: "Pretty Derby Japan"},
	"GameTest_SDGGGE":                {purpose: gaming, platform: "extra:GameTest_SDGGGE", name: "SD Gundam G Generation Eternal"},
	"GameTest_Kancolle":              {purpose: gaming, platform: "extra:GameTest_Kancolle", name: "Kancolle Japan"},
	"MediaUnlockTest_Lemino":         {purpose: video, platform: "extra:MediaUnlockTest_Lemino", name: "Lemino"},
	"MediaUnlockTest_AnimeFesta":     {purpose: video, platform: "extra:MediaUnlockTest_AnimeFesta", name: "AnimeFesta"},
	"MediaUnlockTest_mora":           {purpose: music, platform: "extra:MediaUnlockTest_mora", name: "Mora"},
	"MediaUnlockTest_BBCiPLAYER":     {purpose: video, platform: "extra:MediaUnlockTest_BBCiPLAYER", name: "BBC iPLAYER"},
	"MediaUnlockTest_DisneyPlus":     {purpose: video, platform: "extra:MediaUnlockTest_DisneyPlus", name: "Disney+"},
	"MediaUnlockTest_Dazn":           {purpose: video, platform: "extra:MediaUnlockTest_Dazn", name: "Dazn"},
	"MediaUnlockTest_HuluJP":         {purpose: video, platform: "hulu", name: "Hulu", capability: "JP", regionLabel: "JP"},
	"MediaUnlockTest_MyTVSuper":      {purpose: video, platform: "extra:MediaUnlockTest_MyTVSuper", name: "MyTVSuper"},
	"MediaUnlockTest_NowE":           {purpose: video, platform: "extra:MediaUnlockTest_NowE", name: "Now E"},
	"MediaUnlockTest_ViuTV":          {purpose: video, platform: "extra:MediaUnlockTest_ViuTV", name: "Viu.TV"},
	"MediaUnlockTest_unext":          {purpose: video, platform: "extra:MediaUnlockTest_unext", name: "U-NEXT"},
	"MediaUnlockTest_wowow":          {purpose: video, platform: "extra:MediaUnlockTest_wowow", name: "WOWOW"},
	"MediaUnlockTest_TVer":           {purpose: video, platform: "extra:MediaUnlockTest_TVer", name: "TVer"},
	"MediaUnlockTest_HamiVideo":      {purpose: video, platform: "extra:MediaUnlockTest_HamiVideo", name: "Hami Video"},
	"MediaUnlockTest_4GTV":           {purpose: video, platform: "extra:MediaUnlockTest_4GTV", name: "4GTV.TV"},
	"MediaUnlockTest_SlingTV":        {purpose: video, platform: "extra:MediaUnlockTest_SlingTV", name: "Sling TV"},
	"MediaUnlockTest_PlutoTV":        {purpose: video, platform: "extra:MediaUnlockTest_PlutoTV", name: "Pluto TV"},
	"MediaUnlockTest_HBOMax":         {purpose: video, platform: "extra:MediaUnlockTest_HBOMax", name: "HBO Max"},
	"MediaUnlockTest_Showmax":        {purpose: video, platform: "extra:MediaUnlockTest_Showmax", name: "Showmax"},
	"MediaUnlockTest_Channel4":       {purpose: video, platform: "extra:MediaUnlockTest_Channel4", name: "Channel 4"},
	"MediaUnlockTest_ITVHUB":         {purpose: video, platform: "extra:MediaUnlockTest_ITVHUB", name: "ITV Hub"},
	"MediaUnlockTest_DSTV":           {purpose: video, platform: "extra:MediaUnlockTest_DSTV", name: "DSTV"},
	"MediaUnlockTest_HuluUS":         {purpose: video, platform: "hulu", name: "Hulu", capability: "US", regionLabel: "US"},
	"MediaUnlockTest_encoreTVB":      {purpose: video, platform: "extra:MediaUnlockTest_encoreTVB", name: "encoreTVB"},
	"MediaUnlockTest_Molotov":        {purpose: video, platform: "extra:MediaUnlockTest_Molotov", name: "Molotov"},
	"MediaUnlockTest_LineTVTW":       {purpose: video, platform: "extra:MediaUnlockTest_LineTVTW", name: "LineTV.TW"},
	"MediaUnlockTest_ViuCom":         {purpose: video, platform: "extra:MediaUnlockTest_ViuCom", name: "Viu.com"},
	"MediaUnlockTest_Niconico":       {purpose: video, platform: "extra:MediaUnlockTest_Niconico", name: "Niconico"},
	"MediaUnlockTest_ParamountPlus":  {purpose: video, platform: "extra:MediaUnlockTest_ParamountPlus", name: "Paramount+"},
	"MediaUnlockTest_KKTV":           {purpose: video, platform: "extra:MediaUnlockTest_KKTV", name: "KKTV"},
	"MediaUnlockTest_PeacockTV":      {purpose: video, platform: "extra:MediaUnlockTest_PeacockTV", name: "Peacock TV"},
	"MediaUnlockTest_FOD":            {purpose: video, platform: "extra:MediaUnlockTest_FOD", name: "FOD(Fuji TV)"},
	"MediaUnlockTest_BritBox":        {purpose: video, platform: "extra:MediaUnlockTest_BritBox", name: "BritBox"},
	"MediaUnlockTest_PrimeVideo":     {purpose: video, platform: "prime", name: "Amazon Prime Video", capability: "Access", regionLabel: "Access", regionFirst: true},
	"MediaUnlockTest_Radiko":         {purpose: music, platform: "extra:MediaUnlockTest_Radiko", name: "Radiko"},
	"MediaUnlockTest_DMM":            {purpose: general, platform: "extra:MediaUnlockTest_DMM", name: "DMM (financial site)"},
	"MediaUnlockTest_DMMTV":          {purpose: video, platform: "extra:MediaUnlockTest_DMMTV", name: "DMM TV"},
	"MediaUnlockTest_Catchplay":      {purpose: video, platform: "extra:MediaUnlockTest_Catchplay", name: "CatchPlay+"},
	"MediaUnlockTest_HotStar":        {purpose: video, platform: "extra:MediaUnlockTest_HotStar", name: "HotStar"},
	"MediaUnlockTest_LiTV":           {purpose: video, platform: "extra:MediaUnlockTest_LiTV", name: "LiTV"},
	"MediaUnlockTest_FuboTV":         {purpose: video, platform: "extra:MediaUnlockTest_FuboTV", name: "Fubo TV"},
	"MediaUnlockTest_Fox":            {purpose: video, platform: "extra:MediaUnlockTest_Fox", name: "FOX"},
	"MediaUnlockTest_Joyn":           {purpose: video, platform: "extra:MediaUnlockTest_Joyn", name: "Joyn"},
	"MediaUnlockTest_SpotvNow":       {purpose: video, platform: "extra:MediaUnlockTest_SpotvNow", name: "SPOTV NOW"},
	"MediaUnlockTest_SKY_DE":         {purpose: video, platform: "sky", name: "Sky", capability: "DE", regionLabel: "DE"},
	"MediaUnlockTest_ZDF":            {purpose: video, platform: "extra:MediaUnlockTest_ZDF", name: "ZDF"},
	"MediaUnlockTest_EPIX":           {purpose: video, platform: "extra:MediaUnlockTest_EPIX", name: "MGM+"},
	"MediaUnlockTest_NLZIET":         {purpose: video, platform: "extra:MediaUnlockTest_NLZIET", name: "NLZIET"},
	"MediaUnlockTest_videoland":      {purpose: video, platform: "extra:MediaUnlockTest_videoland", name: "videoland"},
	"MediaUnlockTest_NPO_Start_Plus": {purpose: video, platform: "extra:MediaUnlockTest_NPO_Start_Plus", name: "NPO Start Plus"},
	"MediaUnlockTest_RakutenTV":      {purpose: video, platform: "rakuten_tv", name: "Rakuten TV", capability: "EU", regionLabel: "EU"},
	"MediaUnlockTest_MoviStarPlus":   {purpose: video, platform: "extra:MediaUnlockTest_MoviStarPlus", name: "Movistar+"},
	"MediaUnlockTest_Starz":          {purpose: video, platform: "extra:MediaUnlockTest_Starz", name: "Starz"},
	"MediaUnlockTest_CanalPlus":      {purpose: video, platform: "extra:MediaUnlockTest_CanalPlus", name: "Canal+"},
	"MediaUnlockTest_Sky_CH":         {purpose: video, platform: "sky", name: "Sky", capability: "CH", regionLabel: "CH"},
	"MediaUnlockTest_CBCGem":         {purpose: video, platform: "extra:MediaUnlockTest_CBCGem", name: "CBC Gem"},
	"MediaUnlockTest_AcornTV":        {purpose: video, platform: "extra:MediaUnlockTest_AcornTV", name: "Acorn TV"},
	"MediaUnlockTest_Crave":          {purpose: video, platform: "extra:MediaUnlockTest_Crave", name: "Crave"},
	"MediaUnlockTest_Amediateka":     {purpose: video, platform: "extra:MediaUnlockTest_Amediateka", name: "Amediateka"},
	"MediaUnlockTest_MegogoTV":       {purpose: video, platform: "extra:MediaUnlockTest_MegogoTV", name: "Megogo TV"},
	"MediaUnlockTest_RaiPlay":        {purpose: video, platform: "extra:MediaUnlockTest_RaiPlay", name: "Rai Play"},
	"MediaUnlockTest_TVBAnywhere":    {purpose: video, platform: "extra:MediaUnlockTest_TVBAnywhere", name: "TVBAnywhere+"},
	"GameTest_ProjectSekai":          {purpose: gaming, platform: "extra:GameTest_ProjectSekai", name: "Project Sekai: Colorful Stage"},
	"GameTest_KonosubaFD":            {purpose: gaming, platform: "extra:GameTest_KonosubaFD", name: "Konosuba Fantastic Days"},
	"MediaUnlockTest_NBATV":          {purpose: video, platform: "extra:MediaUnlockTest_NBATV", name: "NBA TV"},
	"MediaUnlockTest_Directv":        {purpose: video, platform: "directv", name: "DIRECTV", capability: "Stream", regionLabel: "Stream"},
	"MediaUnlockTest_SkyGo":          {purpose: video, platform: "sky", name: "Sky", capability: "Go UK", regionLabel: "Go UK"},
	"MediaUnlockTest_DirecTVGO":      {purpose: video, platform: "directv", name: "DIRECTV", capability: "Go", regionLabel: "Go"},
	"MediaUnlockTest_DAM":            {purpose: music, platform: "extra:MediaUnlockTest_DAM", name: "Karaoke@DAM"},
	"MediaUnlockTest_DiscoveryPlus":  {purpose: video, platform: "extra:MediaUnlockTest_DiscoveryPlus", name: "Discovery+"},
	"MediaUnlockTest_ESPNPlus":       {purpose: video, platform: "extra:MediaUnlockTest_ESPNPlus", name: "ESPN+"},
	"MediaUnlockTest_Binge":          {purpose: video, platform: "extra:MediaUnlockTest_Binge", name: "Binge"},
	"MediaUnlockTest_Docplay":        {purpose: video, platform: "extra:MediaUnlockTest_Docplay", name: "Docplay"},
	"MediaUnlockTest_OptusSports":    {purpose: video, platform: "extra:MediaUnlockTest_OptusSports", name: "Optus Sports"},
	"MediaUnlockTest_KayoSports":     {purpose: video, platform: "extra:MediaUnlockTest_KayoSports", name: "Kayo Sports"},
	"MediaUnlockTest_NeonTV":         {purpose: video, platform: "extra:MediaUnlockTest_NeonTV", name: "Neon TV"},
	"MediaUnlockTest_SkyGONZ":        {purpose: video, platform: "sky", name: "Sky", capability: "Go NZ", regionLabel: "Go NZ"},
	"MediaUnlockTest_ThreeNow":       {purpose: video, platform: "extra:MediaUnlockTest_ThreeNow", name: "ThreeNow"},
	"MediaUnlockTest_MaoriTV":        {purpose: video, platform: "extra:MediaUnlockTest_MaoriTV", name: "Maori TV"},
	"MediaUnlockTest_SBSonDemand":    {purpose: video, platform: "extra:MediaUnlockTest_SBSonDemand", name: "SBS on Demand"},
	"MediaUnlockTest_ABCiView":       {purpose: video, platform: "extra:MediaUnlockTest_ABCiView", name: "ABC iView"},
	"MediaUnlockTest_Channel9":       {purpose: video, platform: "extra:MediaUnlockTest_Channel9", name: "Channel 9"},
	"MediaUnlockTest_Telasa":         {purpose: video, platform: "extra:MediaUnlockTest_Telasa", name: "Telasa"},
	"MediaUnlockTest_SetantaSports":  {purpose: video, platform: "extra:MediaUnlockTest_SetantaSports", name: "Setanta Sports"},
	"MediaUnlockTest_Channel5":       {purpose: video, platform: "extra:MediaUnlockTest_Channel5", name: "Channel 5"},
	"MediaUnlockTest_MyVideo":        {purpose: video, platform: "extra:MediaUnlockTest_MyVideo", name: "MyVideo"},
	"MediaUnlockTest_7plus":          {purpose: video, platform: "extra:MediaUnlockTest_7plus", name: "7plus"},
	"MediaUnlockTest_Channel10":      {purpose: video, platform: "extra:MediaUnlockTest_Channel10", name: "Channel 10"},
	"MediaUnlockTest_VideoMarket":    {purpose: video, platform: "extra:MediaUnlockTest_VideoMarket", name: "VideoMarket"},
	"MediaUnlockTest_musicjp":        {purpose: music, platform: "extra:MediaUnlockTest_musicjp", name: "music.jp"},
	"MediaUnlockTest_Popcornflix":    {purpose: video, platform: "extra:MediaUnlockTest_Popcornflix", name: "Popcornflix"},
	"MediaUnlockTest_TubiTV":         {purpose: video, platform: "extra:MediaUnlockTest_TubiTV", name: "Tubi TV"},
	"MediaUnlockTest_Philo":          {purpose: video, platform: "extra:MediaUnlockTest_Philo", name: "Philo"},
	"MediaUnlockTest_FXNOW":          {purpose: video, platform: "extra:MediaUnlockTest_FXNOW", name: "FXNOW"},
	"MediaUnlockTest_Crunchyroll":    {purpose: video, platform: "extra:MediaUnlockTest_Crunchyroll", name: "Crunchyroll"},
	"MediaUnlockTest_CWTV":           {purpose: video, platform: "extra:MediaUnlockTest_CWTV", name: "CW TV"},
	"MediaUnlockTest_Shudder":        {purpose: video, platform: "extra:MediaUnlockTest_Shudder", name: "Shudder"},
	"MediaUnlockTest_TLCGO":          {purpose: video, platform: "extra:MediaUnlockTest_TLCGO", name: "TLC GO"},
	"MediaUnlockTest_Wavve":          {purpose: video, platform: "extra:MediaUnlockTest_Wavve", name: "Wavve"},
	"MediaUnlockTest_Tving":          {purpose: video, platform: "extra:MediaUnlockTest_Tving", name: "Tving"},
	"MediaUnlockTest_CoupangPlay":    {purpose: video, platform: "extra:MediaUnlockTest_CoupangPlay", name: "Coupang Play"},
	"MediaUnlockTest_NaverTV":        {purpose: video, platform: "extra:MediaUnlockTest_NaverTV", name: "Naver TV"},
	"MediaUnlockTest_Afreeca":        {purpose: video, platform: "extra:MediaUnlockTest_Afreeca", name: "Afreeca TV"},
	"MediaUnlockTest_KBSDomestic":    {purpose: video, platform: "extra:MediaUnlockTest_KBSDomestic", name: "KBS Domestic"},
	"MediaUnlockTest_Watcha":         {purpose: video, platform: "extra:MediaUnlockTest_Watcha", name: "WATCHA"},
	"MediaUnlockTest_KOCOWA":         {purpose: video, platform: "extra:MediaUnlockTest_KOCOWA", name: "KOCOWA"},
	"MediaUnlockTest_NBCTV":          {purpose: video, platform: "extra:MediaUnlockTest_NBCTV", name: "NBC TV"},
	"MediaUnlockTest_Crackle":        {purpose: video, platform: "extra:MediaUnlockTest_Crackle", name: "Crackle"},
	"MediaUnlockTest_AETV":           {purpose: video, platform: "extra:MediaUnlockTest_AETV", name: "A&E TV"},
	"MediaUnlockTest_NFLPlus":        {purpose: video, platform: "extra:MediaUnlockTest_NFLPlus", name: "NFL+"},
	"MediaUnlockTest_SkyShowTime":    {purpose: video, platform: "extra:MediaUnlockTest_SkyShowTime", name: "SkyShowTime"},
	"GameTest_MathsSpot":             {purpose: gaming, platform: "extra:GameTest_MathsSpot", name: "MathsSpot Roblox"},
	"MediaUnlockTest_BGlobalSEA":     {purpose: video, platform: "bilibili", name: "Bilibili", capability: "Global SEA", regionLabel: "Global SEA"},
	"MediaUnlockTest_BGlobalTH":      {purpose: video, platform: "bilibili", name: "Bilibili", capability: "Global TH", regionLabel: "Global TH"},
	"MediaUnlockTest_BGlobalID":      {purpose: video, platform: "bilibili", name: "Bilibili", capability: "Global ID", regionLabel: "Global ID"},
	"MediaUnlockTest_BGlobalVN":      {purpose: video, platform: "bilibili", name: "Bilibili", capability: "Global VN", regionLabel: "Global VN"},
	"MediaUnlockTest_AISPlay":        {purpose: video, platform: "extra:MediaUnlockTest_AISPlay", name: "AIS Play"},
	"WebTest_MetaAI":                 {purpose: aiServices, platform: "extra:WebTest_MetaAI", name: "Meta AI", capability: "Web"},
	"WebTest_Wikipedia_Editable":     {purpose: community, platform: "extra:WebTest_Wikipedia_Editable", name: "Wikipedia Editability"},
	"MediaUnlockTest_MeWatch":        {purpose: video, platform: "extra:MediaUnlockTest_MeWatch", name: "MeWatch"},
	"MediaUnlockTest_trueID":         {purpose: video, platform: "extra:MediaUnlockTest_trueID", name: "trueID"},
	"MediaUnlockTest_SonyLiv":        {purpose: video, platform: "extra:MediaUnlockTest_SonyLiv", name: "SonyLiv"},
	"WebTest_EroGameSpace":           {purpose: gaming, platform: "extra:WebTest_EroGameSpace", name: "EroGameSpace"},
	"MediaUnlockTest_DAnimeStore":    {purpose: video, platform: "extra:MediaUnlockTest_DAnimeStore", name: "D Anime Store"},
	"MediaUnlockTest_RakutenTVJP":    {purpose: video, platform: "rakuten_tv", name: "Rakuten TV", capability: "JP", regionLabel: "JP"},
	"MediaUnlockTest_AMCPlus":        {purpose: video, platform: "extra:MediaUnlockTest_AMCPlus", name: "AMC+"},
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
	// Caller-supplied unknown rows stay visible, source-qualified and unguessed.
	return presentation{purpose: general, platform: source + ":" + id}
}

func terminalName(source, id, name string) string {
	if p := presentationFor(source, id); p.name != "" {
		return p.name
	}
	qualified := source + ":" + id
	if name != "" {
		return qualified + " (" + name + ")"
	}
	return qualified
}
