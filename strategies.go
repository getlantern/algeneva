package algeneva

// Strategies is a map of geneva strategies keyed to the country they were found to work in.
//
// Note: China has two sets of strategies, one for hostname censoring and one for keyword censoring. Hostname censor
// strategies are at indices 0-29, and keyword censor strategies start at indices 30.
var Strategies = map[string][]string{
	"China": {
		// hostname censor strategies //
		"[HTTP:version:*]-insert{%09:middle:value:14}-|",
		"[HTTP:path:*]-insert{%20:start:value:1}-|[HTTP:host:*]-duplicate(replace{/:name:64}(replace{/?ultrasurf:value},),)-|",
		"[HTTP:host:*]-duplicate(replace{a:name:64},)-|",
		"[HTTP:method:*]-insert{%20:end:value:1}-|[HTTP:host:*]-duplicate(replace{%2F:name:64},)-|",
		"[HTTP:path:*]-insert{%20:start:value:1}-|[HTTP:host:*]-duplicate(replace{%C2%B0:name:32},)-|",
		"[HTTP:host:*]-insert{%20%0A:start:name:1}-|",
		"[HTTP:host:*]-insert{%20:start:name:1}-|",
		"[HTTP:method:*]-duplicate(,replace{a:name:1407})-|",
		"[HTTP:method:*]-insert{%0A:start:value:4336}-|",
		"[HTTP:method:*]-insert{%20:end:value:1413}-|",
		"[HTTP:method:*]-insert{%20:end:value:1720}-|",
		"[HTTP:path:*]-insert{%0D:end:value:1434}-|",
		"[HTTP:path:*]-insert{%20:start:value:1}-|[HTTP:path:*]-replace{3:value:511}(insert{&:start:value},)-|",
		"[HTTP:path:*]-insert{%3F:start:value:1413}-|",
		"[HTTP:version:*]-insert{%25:middle:value:1434}-|",
		"[HTTP:version:*]-insert{%C3%8B:middle:value:717}-|",
		"[HTTP:method:*]-replace{%3A:value:1}-|",
		"[HTTP:method:*]-replace{HTTP/1.1:value:1}-|",
		"[HTTP:path:*]-insert{%3F:start:value:1}-|",
		"[HTTP:method:*]-insert{%0D:end:value:2}-|",
		"[HTTP:path:*]-insert{%09:start:value:1}-|",
		"[HTTP:path:*]-insert{%0C:start:value:1}-|",
		"[HTTP:path:*]-insert{%0D:start:value:1}-|",
		"[HTTP:path:*]-insert{%20:start:value:1}-|",
		"[HTTP:host:*]-duplicate(replace{%C3%97:name:596},insert{%20:end:name:786})-|",
		"[HTTP:host:*]-replace{%5E:name:926}(duplicate(duplicate(,replace{host:name:1}(insert{%20:start:value:3238},)),),)-|",
		"[HTTP:host:*]-replace{%C3%97:name:1358}(duplicate(duplicate(,replace{host:name:1}(insert{%20:end:value},)),),)-|",
		"[HTTP:host:*]-replace{%C3%97:name:1371}(duplicate(duplicate(,replace{host:name:1}),),)-|",
		"[HTTP:host:*]-replace{PUT:name:423}(duplicate(duplicate(,replace{host:name}),),)-|",
		"[HTTP:version:*]-replace{OPTIONS:value:1}-|",

		// keyword censor strategies //
		"[HTTP:version:*]-insert{%09:middle:value:14}-|",
		"[HTTP:path:*]-insert{%09:end:value:1434}-|[HTTP:path:*]-insert{1:start:value:507}-|",
		"[HTTP:path:*]-insert{%20:end:value:1}-|[HTTP:path:*]-insert{g:end:value:1013}-|",
		"[HTTP:path:*]-insert{%20:start:value:1}-|[HTTP:host:*]-duplicate(replace{/:name:64}(replace{/?ultrasurf:value},),)-|",
		"[HTTP:host:*]-duplicate(replace{a:name:64},)-|",
		"[HTTP:method:*]-insert{%20:end:value:1}-|[HTTP:host:*]-duplicate(replace{%2F:name:64},)-|",
		"[HTTP:path:*]-insert{%20:start:value:1}-|[HTTP:host:*]-duplicate(replace{%C2%B0:name:32},)-|",
		"[HTTP:method:*]-insert{%0A:start:value:4336}-|",
		"[HTTP:path:*]-insert{%0D:end:value:1434}-|",
		"[HTTP:path:*]-insert{%20:start:value:1}-|[HTTP:path:*]-replace{3:value:511}(insert{&:start:value},)-|",
		"[HTTP:version:*]-insert{%25:middle:value:1434}-|",
		"[HTTP:version:*]-insert{%C3%8B:middle:value:717}-|",
		"[HTTP:method:*]-replace{%3A:value:1}-|",
		"[HTTP:method:*]-replace{HTTP/1.1:value:1}-|",
		"[HTTP:path:*]-duplicate(insert{3:middle:value:1004},replace{&ultrasurf:value})-|",
		"[HTTP:method:*]-insert{%0D:end:value:2}-|",
		"[HTTP:path:*]-insert{%0D:start:value:1}-|",
		"[HTTP:host:*]-duplicate(replace{%C3%97:name:596},insert{%20:end:name:786})-|",
		"[HTTP:host:*]-replace{%5E:name:926}(duplicate(duplicate(,replace{host:name:1}(insert{%20:start:value:3238},)),),)-|",
		"[HTTP:host:*]-replace{%C3%97:name:1358}(duplicate(duplicate(,replace{host:name:1}(insert{%20:end:value},)),),)-|",
		"[HTTP:host:*]-replace{%C3%97:name:1371}(duplicate(duplicate(,replace{host:name:1}),),)-|",
		"[HTTP:host:*]-insert{%20:end:value:4081}(duplicate(duplicate(,replace{a:name:1}),insert{%09:start:name:3238}),)-|",
		"[HTTP:host:*]-insert{%20:end:value:4081}(duplicate(duplicate(insert{%09:start:name:3238},),replace{a:name:1}),)-|",
		"[HTTP:host:*]-replace{PUT:name:423}(duplicate(duplicate(,replace{host:name}),),)-|",
		"[HTTP:version:*]-replace{OPTIONS:value:1}-|",
	},
	"India": {
		"[HTTP:host:*]-changecase{lower}-|",
		"[HTTP:host:*]-changecase{upper}-|",
		"[HTTP:version:*]-insert{%09:middle:value:14}-|",
		"[HTTP:path:*]-insert{%09:end:value:1434}-|[HTTP:path:*]-insert{1:start:value:507}-|",
		"[HTTP:path:*]-insert{%20:end:value:1}-|[HTTP:path:*]-insert{g:end:value:1013}-|",
		"[HTTP:method:*]-insert{%09:end:value}-|[HTTP:host:*]-duplicate(replace{a:name:64},)-|",
		"[HTTP:method:*]-insert{%0A:start:value:1}-|[HTTP:host:*]-duplicate(replace{%2F:name:64},)-|",
		"[HTTP:host:*]-duplicate(insert{%0A:end:value:1},)-|",
		"[HTTP:host:*]-duplicate(insert{%0A:random:name:1},)-|",
		"[HTTP:host:*]-duplicate(insert{%20%0A:end:name:1},)-|",
		"[HTTP:host:*]-insert{%09:end:name}-|",
		"[HTTP:host:*]-insert{%0A%0A:start:value:1}-|",
		"[HTTP:host:*]-insert{%0A%20:start:value:1}-|",
		"[HTTP:host:*]-insert{%0A:end:value:1}-|",
		"[HTTP:host:*]-insert{%20%0A:start:name:1}-|",
		"[HTTP:host:*]-insert{%20:end:name:1}-|",
		"[HTTP:host:*]-insert{%20:start:name:1}-|",
		"[HTTP:path:*]-replace{/:value:1434}-|",
		"[HTTP:host:*]-insert{%20:start:value:1413}-|",
		"[HTTP:host:*]-insert{%20:start:value:1434}-|",
		"[HTTP:method:*]-duplicate(,replace{a:name:1407})-|",
		"[HTTP:method:*]-insert{%09:end:value:2568}-|",
		"[HTTP:method:*]-insert{%0A:start:value:4336}-|",
		"[HTTP:method:*]-insert{%20:end:value:1413}-|",
		"[HTTP:method:*]-insert{%20:end:value:1720}-|",
		"[HTTP:path:*]-duplicate(replace{a:name:1}(insert{a:start:value:1408},),)-|",
		"[HTTP:path:*]-insert{%0D:end:value:1434}-|",
		"[HTTP:path:*]-insert{%20:end:value:1413}-|",
		"[HTTP:path:*]-insert{%23:end:value:1413}-|",
		"[HTTP:path:*]-insert{%23:end:value:1}(insert{%C3:end:value:470},)-|",
		"[HTTP:path:*]-insert{%3F:end:value:1413}-|",
		"[HTTP:path:*]-insert{%3F:start:value:1413}-|",
		"[HTTP:path:*]-replace{/:value:1414}-|",
		"[HTTP:version:*]-insert{%20:end:value:1434}-|",
		"[HTTP:version:*]-insert{%20:start:value:1434}-|",
		"[HTTP:version:*]-insert{%25:middle:value:1434}-|",
		"[HTTP:version:*]-insert{%C2%81:end:value:773}-|",
		"[HTTP:version:*]-insert{%C3%8B:middle:value:717}-|",
		"[HTTP:method:*]-replace{%3A:value:1}-|",
		"[HTTP:method:*]-duplicate(,)-|",
		"[HTTP:method:*]-replace{HTTP/1.1:value:1}-|",
		"[HTTP:path:*]-duplicate(insert{3:middle:value:1004},replace{&ultrasurf:value})-|",
		"[HTTP:path:*]-insert{%3F:start:value:1}-|",
		"[HTTP:method:*]-insert{%09:end:value:1}-|",
		"[HTTP:method:*]-insert{%09:start:value:1}-|",
		"[HTTP:method:*]-insert{%0A:start:value:1}-|",
		"[HTTP:method:*]-insert{%0B:end:value:1}-|",
		"[HTTP:method:*]-insert{%0D:end:value:2}-|",
		"[HTTP:path:*]-insert{%09:end:value:1}-|",
		"[HTTP:path:*]-insert{%09:start:value:1}-|",
		"[HTTP:path:*]-insert{%0C:start:value:1}-|",
		"[HTTP:path:*]-insert{%0D:start:value:1}-|",
		"[HTTP:path:*]-insert{%20:end:value:1}-|",
		"[HTTP:version:*]-insert{%0A%09%0A%09:end:value:1}-|",
		"[HTTP:version:*]-insert{%0A%20%0A%20:end:value:1}-|",
		"[HTTP:version:*]-insert{%20%0A%09:end:value:1}-|",
		"[HTTP:version:*]-insert{%20:end:value:1}-|",
		"[HTTP:host:*]-duplicate(replace{%C3%97:name:596},insert{%20:end:name:786})-|",
		"[HTTP:host:*]-replace{%C3%97:name:1358}(duplicate(duplicate(,replace{host:name:1}(insert{%20:end:value},)),),)-|",
		"[HTTP:host:*]-replace{%C3%97:name:1371}(duplicate(duplicate(,replace{host:name:1}),),)-|",
		"[HTTP:host:*]-replace{PUT:name:423}(duplicate(duplicate(,replace{host:name}),),)-|",
		"[HTTP:version:*]-replace{OPTIONS:value:1}-|",
		"[HTTP:version:*]-duplicate-|",
	},
	"Kazakhstan": {
		"[HTTP:method:*]-insert{%09:end:value}-|[HTTP:host:*]-duplicate(replace{a:name:64},)-|",
		"[HTTP:method:*]-insert{%0A:start:value:1}-|[HTTP:host:*]-duplicate(replace{%2F:name:64},)-|",
		"[HTTP:host:*]-insert{%09:end:name}-|",
		"[HTTP:host:*]-insert{%09:end:value:1}-|",
		"[HTTP:host:*]-insert{%09:start:value:1}-|",
		"[HTTP:host:*]-insert{%0A%0A:start:value:1}-|",
		"[HTTP:host:*]-insert{%0A%20:start:value:1}-|",
		"[HTTP:host:*]-insert{%20%0A:start:name:1}-|",
		"[HTTP:host:*]-insert{%20:end:name:1}-|",
		"[HTTP:host:*]-insert{%20:end:value:1}-|",
		"[HTTP:host:*]-insert{%20:start:name:1}-|",
		"[HTTP:host:*]-insert{%20:start:value:1434}-|",
		"[HTTP:method:*]-insert{%09:end:value:2568}-|",
		"[HTTP:method:*]-insert{%0A:start:value:4336}-|",
		"[HTTP:method:*]-insert{%20:end:value:1720}-|",
		"[HTTP:version:*]-insert{%20:end:value:1434}-|",
		"[HTTP:version:*]-insert{%20:start:value:1434}-|",
		"[HTTP:version:*]-insert{%25:middle:value:1434}-|",
		"[HTTP:version:*]-insert{%C2%81:end:value:773}-|",
		"[HTTP:version:*]-insert{%C3%8B:middle:value:717}-|",
		"[HTTP:method:*]-replace{%3A:value:1}-|",
		"[HTTP:method:*]-duplicate(,)-|",
		"[HTTP:method:*]-replace{HTTP/1.1:value:1}-|",
		"[HTTP:method:*]-insert{%09:end:value:1}-|",
		"[HTTP:method:*]-insert{%09:start:value:1}-|",
		"[HTTP:method:*]-insert{%0A:start:value:1}-|",
		"[HTTP:method:*]-insert{%0B:end:value:1}-|",
		"[HTTP:method:*]-insert{%0D:end:value:2}-|",
		"[HTTP:version:*]-insert{%0A%09%0A%09:end:value:1}-|",
		"[HTTP:version:*]-insert{%0A%09:end:value:1}-|",
		"[HTTP:version:*]-insert{%0A%20%0A%20:end:value:1}-|",
		"[HTTP:version:*]-insert{%20%0A%09:end:value:1}-|",
		"[HTTP:host:*]-duplicate(replace{%C3%97:name:596},insert{%20:end:name:786})-|",
		"[HTTP:host:*]-replace{%5E:name:926}(duplicate(duplicate(,replace{host:name:1}(insert{%20:start:value:3238},)),),)-|",
		"[HTTP:host:*]-replace{%C3%97:name:1358}(duplicate(duplicate(,replace{host:name:1}(insert{%20:end:value},)),),)-|",
		"[HTTP:host:*]-insert{%20:end:value:4081}(duplicate(duplicate(,replace{a:name:1}),insert{%09:start:name:3238}),)-|",
		"[HTTP:host:*]-insert{%20:end:value:4081}(duplicate(duplicate(insert{%09:start:name:3238},),replace{a:name:1}),)-|",
	},
}
