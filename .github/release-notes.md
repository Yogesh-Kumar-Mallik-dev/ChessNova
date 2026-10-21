## ChessPasant (v0.1.7)

Welcome to the official cross-platform release of **ChessPasant** — a modern, distraction-free chess platform and tactical analysis laboratory engineered with **Rust** and **Tauri v2**.

---

### Official Download Packages

| Platform | Package | Architecture | Target |
| :--- | :--- | :--- | :--- |
| <img src="https://raw.githubusercontent.com/Yogesh-Kumar-Mallik-dev/ChessPasant/main/apps/web/static/icons/archlinux.svg" width="16" height="16" valign="middle" /> **Arch Linux** | `chesspasant-0.1.7-1-x86_64.pkg.tar.zst` | x86_64 | Pacman (`sudo pacman -U`) |
| <img src="https://raw.githubusercontent.com/Yogesh-Kumar-Mallik-dev/ChessPasant/main/apps/web/static/icons/archlinux.svg" width="16" height="16" valign="middle" /> **Arch Linux / AUR** | `ChessPasant_0.1.7_x86_64.tar.gz` | x86_64 | Standalone / `makepkg` |
| <img src="https://raw.githubusercontent.com/Yogesh-Kumar-Mallik-dev/ChessPasant/main/apps/web/static/icons/tux.svg" width="16" height="16" valign="middle" /> **Linux (Universal)** | `ChessPasant_0.1.7_amd64.AppImage` | x86_64 / amd64 | Portable (All distributions) |
| <img src="https://raw.githubusercontent.com/Yogesh-Kumar-Mallik-dev/ChessPasant/main/apps/web/static/icons/tux.svg" width="16" height="16" valign="middle" /> **Linux (Debian/Ubuntu)** | `chesspasant_0.1.7_amd64.deb` | x86_64 / amd64 | APT / dpkg |
| <img src="https://cdn.jsdelivr.net/npm/simple-icons@latest/icons/windows11.svg" width="16" height="16" valign="middle" /> **Windows** | `ChessPasant_0.1.7_x64-setup.exe` | x64 | Windows 10 & 11 Setup |
| <img src="https://cdn.jsdelivr.net/npm/simple-icons@latest/icons/windows11.svg" width="16" height="16" valign="middle" /> **Windows** | `ChessPasant_0.1.7_x64_en-US.msi` | x64 | Enterprise MSI Installer |
| <img src="https://cdn.jsdelivr.net/npm/simple-icons@latest/icons/apple.svg" width="16" height="16" valign="middle" /> **macOS** | `ChessPasant_0.1.7_universal.dmg` | Universal Binary | Apple Silicon (M1/M2/M3/M4) & Intel |
| <img src="https://cdn.jsdelivr.net/npm/simple-icons@latest/icons/android.svg" width="16" height="16" valign="middle" /> **Android** | `ChessPasant_0.1.7.apk` | arm64-v8a / aarch64 | Direct APK Installer |
| <img src="https://cdn.jsdelivr.net/npm/simple-icons@latest/icons/apple.svg" width="16" height="16" valign="middle" /> **iOS** | `ChessPasant_0.1.7_unsigned.ipa` | arm64 | Sideload / AltStore / TrollStore |

---

### Highlights in this Release
- Dual Stockfish 19 Engine: Integrated native Stockfish 19 UCI engine with baseline depth 14 search (depth 16 endgame scaling) and client-side Stockfish 19 Lite WASM with embedded NNUE neural network (`nn-61e7af4bb97d.nnue`).
- Grandmaster Tactical Review Coach: Real-time evaluation curves with tactical motif classification (Forks, Pins, Hanging Blunders, Discovered Checks, Queen Trades, Sacrifices, and Missed Wins).
- Ultra-Lightweight & Blazing Fast: Engineered with Rust and native webview components, consuming under 60 MB RAM compared to heavy Electron alternatives.
- Native Window Manager & Tiling Integration: Seamlessly integrates with system window managers and tiling setups (Hyprland, Sway, i3) with native OS controls on Windows and macOS.
- 100% Offline Capable: Play against AI bots, enjoy local pass-and-play, and analyze games with zero internet connection required.

---

### Installation Notes
- **Arch Linux**: Run `sudo pacman -U chesspasant-*.pkg.tar.zst` or extract `ChessPasant_*_x86_64.tar.gz` and run `./install.sh`.
- **Linux AppImage**: Run `chmod +x ChessPasant_*.AppImage && ./ChessPasant_*.AppImage`.
- **Linux DEB**: Run `sudo dpkg -i chesspasant_*.deb` or double-click to install via your software center.
- **Windows**: Launch the `.exe` or `.msi`. If Windows Defender SmartScreen displays a warning, click **More info** -> **Run anyway**.
- **macOS**: Mount the `.dmg` and drag ChessPasant into your `Applications` folder. On first launch, right-click (or Control-click) and select **Open** to approve the Gatekeeper security prompt.
- **Android**: Download `ChessPasant_*.apk` and tap to install on your Android device (allow "Install unknown apps" if prompted).
- **iOS**: Sideload `ChessPasant_*_unsigned.ipa` using AltStore, SideStore, Sideloadly, TrollStore, or install via Xcode.
