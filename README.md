# HateKey

HateKey is a keyboard mapping project aimed at PC gamers who spend every day hammering the same key for rapid repeated taps (clicking the same key many times quickly).

Especially those playing TH2L—my output efficiency playing Seiko on PC might be worse than playing Seiko on a phone; my fingers are about to give out, and my output frequency may not even be higher than that of mobile players. Some PC players might feel a bit frustrated—I'm one of them.

HateKey allows you to combine two physical keys into one logical key, used directly as an interaction key.

Of course, this might cause you a bit of minor trouble in work scenarios—just close the exe after gaming. Minimizing the window won't stop you from using it.

---

## Dependencies

- Go 1.22+
- `golang.org/x/sys`

---

## JSON Usage Tutorial

### Configuration File

`json\keymacro.json`, relative to the directory where `hate.exe` is located. Restart the exe after changes for them to take effect.

### Basic Structure

```json
{
    "keys:mapping": {
        "z": ["z", "x"]
    }
}
```

· `keys:mapping` is fixed.
· Each key-value pair is a key group: the group name is the logical key, and the array contains the member keys.

Key Group Meaning

`"z": ["z", "x"]`:

· Press x → output z
· Press z → output z as-is

· Pressing a member key → outputs the logical key
· The logical key itself is not mapped
· A physical key can only appear in one group, otherwise the mapping target is uncertain

Supported Key Names

· Letters a–z (case-insensitive)
· Numbers 0–9
· space, enter, esc, tab, shift, ctrl, alt
· f1–f12

Multiple Groups Example

```json
{
    "keys:mapping": {
        "z": ["a", "x"],
        "q": ["c", "v"],
        "1": ["n", "m"]
    }
}
```

Effect: a/x → z, c/v → q, n/m → 1. Unlimited number of groups, as long as members don't repeat.

Notes

· JSON must be UTF-8 without BOM, otherwise parsing will fail.
· If the configuration is incorrect or hook installation fails, the program exits silently. Make sure `json\keymacro.json` exists and run as administrator.
· When moving the exe program, be sure to take the json folder with it!
