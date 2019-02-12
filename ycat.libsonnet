// Usefull functions
// Completely inefficient :) should be ported to nativeFuncs
local result(input, arr) =
    if std.isString(input) && std.isArray(arr) then std.join('', arr) else arr
    ;

local has(x, y) = 
    local t = std.type(x);
    if t == 'array' then
        std.count(x, y) > 0
    else if t == 'object' then
        std.objectHas(x, y)
    else if t == 'string' then
        std.length(std.findSubstr(y, x)) > 0
    else
        false
    ;

local skipFunc(x) = if std.type(x) == 'function' then x else function(y) y == x;
local trimFunc(cutset) =
    if std.isString(cutset) then
        local cs = std.stringChars(cutset);
        function (c) std.count(cs, c) > 0
    else if std.isArray(cutset) then
        function (c) std.count(cutset, c) > 0
    else if std.isFunction(cutset) then
        cutset
    else if std.isObject(cutset) then
        function (c) std.objectHas(cutset, c)
    else if std.isNumber(cutset) then
        function (c) std.codepoint(c) == cutset
    else
        function (c) false
    ;
std + {
    local _ = self
    , len:: std.length
    , has:: has
    , get(obj, key, v=null)::
        if std.isObject(obj) && std.objectHas(obj, key) then obj[key] else v
    , sum(arr)::
        local add(total, n) = total + n;
        std.foldl(add, arr, 0)
    , avg(arr)::
        local n = std.length(arr);
        if n > 0 then _.sum(arr)/n else 0
    , skipWhile(pred, arr)::
        local func = skipFunc(pred);
        local skip = function(acc, x) {
            skip:: if acc.skip then func(x) else false,
            out:: if self.skip then [] else acc.out + [x],
        };
        result(arr, std.foldl(skip, arr, {skip:: true, out:: []}).out)
    , takeWhile(pred, arr)::
        local func = skipFunc(pred);
        local take = function(acc, x) {
            ok:: if acc.ok then func(x) else false,
            out:: if self.ok then acc.out + [x] else acc.out,
        };
        result(arr, std.foldl(take, arr, {ok:: true, out:: []}).out)
    , indexOf(arr, x)::
        local fn(y) = x != y;
        local n = std.length(_.takeWhile(fn, arr));
        if n == std.length(arr) then -1 else n
    , not(func):: function(x) if func(x) then false else true
    , takeUntil(pred, arr):: _.takeWhile(_.not(skipFunc(pred)), arr)
    , skipUntil(pred, arr):: _.skipWhile(_.not(skipFunc(pred)), arr)
    , trunc(arr, size):: // Truncate array
        local sz = std.min(size, std.length(arr));
        result(arr, std.makeArray(sz, function(i) arr[i]))
    , rev(arr):: // Reverse array
        local size = std.length(arr);
        local n = size - 1;
        result(arr, std.makeArray(size, function(i) arr[n-i]))
    , ascii:: {
        local inRange(min, max) =
            local _min = std.codepoint(min);
            local _max = std.codepoint(max);
            function (c) _min <= std.codepoint(c) && std.codepoint(c) <= _max
        , isLower:: inRange('a', 'z')
        , isUpper:: inRange('A', 'Z')
        , isDigit:: inRange('0', '9')
        , space:: " \n\t\r"
        , isAlpha(c):: _.ascii.isLower(c) || _.ascii.isUpper(c)
        , isAlnum(c):: _.ascii.isLower(c) || _.ascii.isUpper(c) || _.ascii.isDigit(c)
        , isSpace(c):: c == " " || c == "\n" || c == "\t" || c == "\r"
    }
    , squeeze(s, cutset)::
        local tr = trimFunc(cutset);
        local fn(acc, c) =
            local n = std.length(acc) - 1;
            if tr(c) && n >= 0 && tr(acc[n]) then
                acc
            else
                acc + [c];
        local ss = std.foldl(fn, s, []);
        result(s, ss)

    , normalize(s):: // Trim and consolidate sequential whitespace to ' '
        local toSpace(c) = if _.ascii.isSpace(c) then ' ' else c;
        local ls = _.skipWhile(" ", _.map(toSpace, s));
        local rs = _.skipWhile(" ", _.rev(ls));
        local ss = _.squeeze(_.rev(rs), " ");
        result(s, ss)

    , trimLeft(s, cutset=_.ascii.space):: // Trim left side of a string
        local tr = trimFunc(cutset);
        _.skipWhile(tr, s)

    , trimRight(s, cutset=_.ascii.space):: // Trim right side of a string
        local tr = trimFunc(cutset);
        local rs = _.skipWhile(tr, _.rev(s));
        local ls = _.rev(rs);
        result(s, ls)
    , trim(s, cutset=_.ascii.space):: // Trim both sides of a string
        local tr = trimFunc(cutset);
        local rs = _.skipWhile(tr, _.rev(s));
        local ls = _.skipWhile(tr, _.rev(rs));
        result(s, ls)
    , k8s:: {
        maxNameSize:: 253
        , trunc(name)::
            if std.length(name) > _.k8s.maxNameSize then
                result(name, _.trunc(name, _.k8s.maxNameSize))
            else
                name
        , namespace(res, ns, override=true)::
            local n = _.k8s.name(ns);
            if override then
                res + {metadata: {namespace: n}}
            else
                {metadata+: {namespace: n}} + res
        , name(s):: // convert string to kubernetes name
            local fn(c) =
                if _.ascii.isLower(c) then c
                else if _.ascii.isDigit(c) then c
                else if _.ascii.isUpper(c) then std.asciiLower(c)
                else '-';
            local cs = std.map(fn, s);
            local rs = _.skipWhile('-', _.rev(cs)); // trim - from end
            local ls = _.skipUntil(_.ascii.isLower, _.rev(rs)); // trim -,0-9 from start
            local name = _.squeeze(ls, "-"); // squeeze sequential '-'
            result(s, _.k8s.trunc(name))
    }

}