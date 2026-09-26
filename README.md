## 這個目錄架構的三大實戰原則：
無 Fyne 污染的 internal/ (最重要)：
internal/ 資料夾裡的程式碼應該純粹只做資料運算或網路通訊。這裡面絕對不要出現 import "fyne.io/fyne/v2"。 這樣能確保你的商業邏輯與 UI 框架完全脫鉤，未來如果想把軟體轉成網頁版或換成 Wails，底層邏輯可以 100% 沿用。

讓 main.go 保持極簡：
不要把所有的視窗佈局都塞在 main.go 裡。main.go 的職責只有一個：呼叫 internal 讀取設定、把資料傳給 ui/screens/ 畫出首頁，然後執行 app.Run()。

靜態資源全部打包進程式碼：
桌面軟體最怕使用者遺失旁邊的圖片檔。Fyne 提供了極度好用的指令 fyne bundle。平常把圖片放在 assets/ 下，編譯前跑一次工具，它會把圖片轉成 bundle/bundled.go 裡的 byte 陣列。最終發布時，你依然只會產出一個乾淨無比的單一執行檔（.exe 或 macOS 的 .app）。

## 溝通策略

| 行為 | 一般認知 | 策略 | 具體怎麼說 |
| --- | --- | --- | --- |
| 約會遲到，無故消失 | 懶惰，不重視對方 | 不能到場，提前通知，不要勉強自己赴約。 可以說：我明天可能會約你，我5點跟你說/ | 今天：9/26 landan，hi 我明天 9/27 可能可以跟你約會， 如果 我 9/27 5pm 消息給你，就是可以，如果我沒有傳訊息給你，那就是我需要休息，不好意思，我真的比較困難。 謝謝你。 或是我們可以約 9/28 4:00pm ，我剛下班我應該可以。 |
| 聊天要錢要錢 | 覺得賺錢很容易，金錢隨便 | 約出來對方，單獨的時間，請對方吃麥當勞， 講述原因。 | hi，landan 之前 8/1 我有跟你說，我從新工作了。我沒有客人了， 因為我是剛回到崗位，太久沒工作了。 我非常擔心我的工作，然後我希望你可以在 8/1 開始去幫助我。不好意思呢，給你造麻煩了。 現在 8/25 ，其實我的手頭有點緊張，我的家人過世了， 需要一大筆費用，這個是突發的狀況， 我知道我 在 8/1 跟你說你來 jtv，就不用給我 金錢了，這個我記得， 只是我真的沒有辦法了，我壓力很大。請問您能不能 提早把 9/25 的津貼給我呢，不好意思 我沒有其他辦法了。 |
| 封鎖對方 | 拒絕溝通，非常無理 | 不要回覆，說3天後我再跟你談，如果3天後還是不行，提前1天說。 | 9/25 這件事情，我還是沒有共識， 我覺得現在大家都在情緒上，不好溝通。 能不能 9/27 再溝通。不要擔心，我一定會給你時間 跟發言的機會。或是你可以來的店裡溝通。 |


Go 做桌面程式的缺點， go 對 desktop API 支援偏低，譬如 讀取usb 視訊，需要透過 ffmpeg，或用 cgo 呼叫系統框架(macOS 用 AVFoundation)。本專案原本用 ffmpeg，現在是後者，見 pkg/avfoundation/，CPU 比較見下方
Rust 做桌面程式的缺點，rust 需要學習新語言
Node 做桌面程式的缺點，體積太大
Python 做桌面程式的缺點。python 比較拿不到全部os 的性能

時程第一：選 python
性能第一：選 Rust
想要學習C：選golang
開發體驗全JS：選 Node

## 讀攝影機：ffmpeg 拉訊號 vs Objective-C 拉訊號的 CPU 比較

原本用 `ffmpeg -f avfoundation` 讀攝影機，從 stdout 收 MJPEG 再解碼顯示；現在改成用 Objective-C 直接呼叫 AVFoundation（`pkg/avfoundation/`）。

**測試條件（2026-09-26）**

- Apple M4（10 核心）、macOS 26.5、ffmpeg 8.1.2、內建「MacBook Air相機」
- 擷取 1280x720 @ 30fps；錄影 4Mbps H.264（兩邊都是硬體編碼）；預覽 640x360 @ 15fps
- 每 2 秒一張完整尺寸截圖（正式設定是 600 秒，這裡調短是為了讓截圖也被量到）
- 跑 12 秒。百分比以「一顆核心 = 100%」計，10 核心的機器上 45% 約等於整機 4.5%
- 系統的「視訊效果 > 手勢反應」當時是**開著**的

**結果**

| | 做法 | 12 秒 CPU 時間 | 換算 |
| --- | --- | --- | --- |
| ffmpeg | `ffmpeg -f avfoundation`（nv12）→ 預覽 MJPEG + mp4 + 截圖，只算 ffmpeg 行程 | user 4.43s + sys 1.04s = 5.47s | 約 41~46% |
| Objective-C | `pkg/avfoundation`，只算測試行程 | BGRA 版 44.8%，改成 YUV 版 43.3% | 約 43~45% |

ffmpeg 的牆鐘時間是 13.5 秒（含啟動），除以 12 或 13.5 會差一點，所以寫成範圍。

**結論：同樣負載下 CPU 差不多，沒有明顯贏家。** Objective-C 版的好處不在 CPU：

- 不用安裝 ffmpeg，不用解析 stdout 的 MJPEG，預覽和截圖直接拿到像素，不經過 JPEG 壓縮再解壓（ffmpeg 版 App 端另外要 JPEG 解碼並渲染，之前量到約 11%，這次沒重量）
- 權限、相機中斷這類事件可以直接處理，錯誤訊息比 ffmpeg 的 stderr 好用

**CPU 花在哪**（對 Objective-C 版取樣 5 秒，`sample <pid> 5`，看不在等待的執行緒）

- 大宗是**系統的視訊效果**，不是我們的程式碼：`CoreMLNNProcessor`、`VCPHandGestureVideo`、`VN.detector`、`FigObjectDetection`，合計超過一半，是手勢反應偵測在跑 ML。
- 我們自己的 `camera-stream` 佇列約 15%：YUV 轉 RGBA、預覽縮放、Go 端複製像素、JPEG 編碼。
- mp4 錄影用硬體編碼，幾乎不占 CPU。
- 想再降 CPU，先動系統設定：控制中心 > 視訊效果 > 關掉「反應」。這是全系統設定，ffmpeg 版也會受影響。

**這份量測的限制**

- 每種只量**一次**，沒有重複取樣，單次結果有誤差。
- 兩邊都只量「擷取這一段」：Objective-C 版是在測試裡量的，ffmpeg 版只算 ffmpeg 行程，都沒含 Fyne 的 OpenGL 渲染，不是完整 App 的實測。
- **和舊紀錄對不上**：更早一次調校記錄顯示 ffmpeg 版 App 約 11% + ffmpeg 約 17.8%，合計約 29%；今天同樣參數量到的 ffmpeg 是 41~46%。原因沒查出來（可能是當時手勢反應沒開、光線不同、或量測方法不同）。所以那個 29% **不要**拿來跟今天的 Objective-C 數字比。
- 攝影機實際幀率會隨光線變動（這台約 27~28fps，不是 30），會影響 CPU。

**怎麼重現**

ffmpeg 版（`-t` 要放在 `-i` 前面，放在輸出前面只會限制第一個輸出）：

```sh
/usr/bin/time -p ffmpeg -hide_banner -loglevel error \
  -f avfoundation -framerate 30 -video_size 1280x720 -pixel_format nv12 -t 12 -i default:none \
  -vf scale=640:360 -r 15 -f image2pipe -vcodec mjpeg -q:v 5 - \
  -c:v h264_videotoolbox -b:v 4M -movflags +frag_keyframe+empty_moov -y /tmp/rec.mp4 \
  -vf fps=1/2 -q:v 2 -f image2 -strftime 1 "/tmp/s-%H%M%S.jpg" > /dev/null
```

Objective-C 版：寫一個暫時的測試呼叫 `avfoundation.Stream`（參數同上，`SnapshotEvery` 設 2 秒，跑 12 秒），前後各用 `syscall.Getrusage` 取 user + sys 時間，除以牆鐘時間，跑完刪掉測試。

**實作備註**（`pkg/avfoundation/`）

- 只有 macOS + cgo 能用；`.m` / `.h` 放在 Go 檔同一個目錄，是 cgo 的要求。
- macOS 的 session 啟動時會套用 preset，可能讓相機停在自己的預設格式（這台是 1920x1080），而 `AVCaptureSessionPresetInputPriority` 在 macOS 不存在。所以改成 `startRunning` 之後用 `activeFormat` 選剛好 1280x720、支援目標幀率的格式；切換前可能還有幾張舊尺寸的畫面，程式會丟掉。找不到符合的格式就退回最接近的 preset。
- 相機輸出原生 YUV（420v），錄影直接餵硬體編碼器；只有預覽和截圖才用 vImage 轉成 RGBA。
- 截圖等 10 張畫面之後才拍，因為剛開相機時自動曝光還沒穩，畫面偏暗。
- 第一次會跳出攝影機權限對話框，權限歸給啟動這個程式的終端機（跟 ffmpeg 時一樣）。
