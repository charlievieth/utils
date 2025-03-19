#!/bin/sh
set -e

API_KEY="${ECOWITT_API_KEY}"
APP_KEY="${ECOWITT_APPLICATION_KEY}"
APP_MAC="${ECOWITT_MAC}"
curl "https://api.ecowitt.net/api/v3/device/info?application_key=${APP_KEY}&api_key=${API_KEY}&mac=${APP_MAC}&call_back=all"
