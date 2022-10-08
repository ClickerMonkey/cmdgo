Interactive (no arguments)
```
> go run main.go profile
Your name: ClickerMonkey
Your age: 33
Favorite numbers: 1
More? (y/n): y
Favorite numbers: 3
More? (y/n): y
Favorite numbers: 7
More? (y/n): y
Favorite numbers: 9
More? (y/n): n
Do you have any favorite movies? (y/n): y
Title: Lord of the Rings
Rating (0-10): 10
More? (y/n): y
Title: Matrix
Rating (0-10): 8.67
More? (y/n): n

Profile: {"Name":"ClickerMonkey","Age":33,"FaveNumbers":[1,3,7,9],"FaveMovies":[{"Title":"Lord of the Rings","Rating":10},{"Title":"Matrix","Rating":8.67}]}
```

Arguments only
```
> go run main.go profile --name ClickerMonkey --age 33 --favenum 1 --favenum 3 --favenum 7 --favenum 9 --movies-1-title "Lord of the Rings" --movies-1-rating 10 --movies-2-title Matrix --movies-2-rating 8.67

Profile: {"Name":"ClickerMonkey","Age":33,"FaveNumbers":[1,3,7,9],"FaveMovies":[{"Title":"Lord of the Rings","Rating":10},{"Title":"Matrix","Rating":8.67}]}
```

YAML
```
> go run main.go profile --yaml profile.yaml

Profile: {"Name":"ClickerMonkey","Age":33,"FaveNumbers":[1,3,7,9],"FaveMovies":[{"Title":"Lord of the Rings","Rating":10},{"Title":"Matrix","Rating":8.67}]}
```

JSON
```
> go run main.go profile --json profile.json

Profile: {"Name":"ClickerMonkey","Age":33,"FaveNumbers":[1,3,7,9],"FaveMovies":[{"Title":"Lord of the Rings","Rating":10},{"Title":"Matrix","Rating":8.67}]}
```

XML
```
> go run main.go profile --xml profile.xml

Profile: {"Name":"ClickerMonkey","Age":33,"FaveNumbers":[1,3,7,9],"FaveMovies":[{"Title":"Lord of the Rings","Rating":10},{"Title":"Matrix","Rating":8.67}]}
```