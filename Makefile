.PHONY: all console graphique app test clean

all: console app

console:
	go build -o logitech-dual-action .

graphique:
	go build -tags gui -o logitech-dual-action-graphique .

app: graphique
	mkdir -p "Logitech Dual Action.app/Contents/MacOS"
	cp logitech-dual-action-graphique "Logitech Dual Action.app/Contents/MacOS/logitech-dual-action-graphique"
	cp app-Info.plist "Logitech Dual Action.app/Contents/Info.plist"
	codesign --force --sign - "Logitech Dual Action.app"

test:
	go test ./...
	go test -tags gui ./...
	go vet ./...
	go vet -tags gui ./...

clean:
	rm -f logitech-dual-action logitech-dual-action-graphique
	rm -rf "Logitech Dual Action.app"
