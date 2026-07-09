#!/usr/bin/env bash
#
# Yerel (offline) ses→metin kurulumu — whisper.cpp:
#   1. whisper.cpp CLI'ını (`whisper-cli`) kurar
#   2. Türkçe destekli çok dilli bir ggml modeli indirir
#
# whisper.cpp tamamen cihaz üstünde çalışır — ses dışarı çıkmaz, API gerekmez.
#
# Kullanım:  scripts/setup_whisper.sh [model]
#   model: tiny | base | small (varsayılan) | medium
#          büyük = daha isabetli ama yavaş. "small" iyi bir denge.
set -euo pipefail

MODEL="${1:-small}"
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MODELS_DIR="$REPO_ROOT/models"
MODEL_FILE="$MODELS_DIR/ggml-${MODEL}.bin"
MODEL_URL="https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-${MODEL}.bin"

echo "==> whisper.cpp kurulumu — model: $MODEL"

# 1. whisper-cli binary
if command -v whisper-cli >/dev/null 2>&1; then
  echo "    whisper-cli zaten kurulu: $(command -v whisper-cli)"
elif command -v brew >/dev/null 2>&1; then
  echo "    whisper-cpp Homebrew ile kuruluyor..."
  brew install whisper-cpp
else
  echo "!!  Homebrew yok. whisper.cpp'yi elle kurup WHISPER_BIN ayarlayın."
  echo "    https://github.com/ggml-org/whisper.cpp"
  exit 1
fi

# ffmpeg (webm/opus → 16kHz mono WAV dönüşümü için gerekli)
if ! command -v ffmpeg >/dev/null 2>&1; then
  if command -v brew >/dev/null 2>&1; then
    echo "    ffmpeg kuruluyor..."
    brew install ffmpeg
  else
    echo "!!  ffmpeg yok. 'brew install ffmpeg' ile kurun."
  fi
fi

# 2. Model indir (varsa atla)
mkdir -p "$MODELS_DIR"
if [ -f "$MODEL_FILE" ]; then
  echo "    model zaten var: $MODEL_FILE"
else
  echo "    model indiriliyor -> $MODEL_FILE (birkaç yüz MB, tek seferlik)"
  curl -L --fail --progress-bar -o "$MODEL_FILE" "$MODEL_URL"
fi

echo ""
echo "==> Tamam. .env dosyanıza ekleyin (mutlak yol en güvenlisi):"
echo ""
echo "    WHISPER_BIN=whisper-cli"
echo "    WHISPER_MODEL=$MODEL_FILE"
echo "    WHISPER_LANG=tr"
echo ""
echo "    Sonra sunucuyu yeniden başlatın."
