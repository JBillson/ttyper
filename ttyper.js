#!/usr/bin/env node
'use strict';
const fs = require('fs');
const path = require('path');
const os = require('os');

// ─── ANSI helpers ────────────────────────────────────────────────────────────
const ESC = '\x1b[';
let _buf = '';
const ansi = {
  clear:         () => { _buf = ESC + '2J' + ESC + 'H'; },
  hide:          () => { _buf += ESC + '?25l'; },
  show:          () => { _buf += ESC + '?25h'; },
  moveTo:        (r, c) => { _buf += ESC + `${r};${c}H`; },
  clearLine:     () => { _buf += ESC + '2K'; },
  write:         (s) => { _buf += s; },
  flush:         () => { process.stdout.write(_buf); _buf = ''; },
};

const C = {
  reset:   '\x1b[0m',
  bold:    '\x1b[1m',
  dim:     '\x1b[2m',
  correct: '\x1b[32m',   // green
  wrong:   '\x1b[31m',   // red
  cursor:  '\x1b[7m',    // reverse (cursor block)
  pending: '\x1b[90m',   // dark grey (untyped)
  accent:  '\x1b[36m',   // cyan
  yellow:  '\x1b[33m',
  white:   '\x1b[97m',
};

// ─── Word lists ───────────────────────────────────────────────────────────────
const WORD_LISTS = {
  common: [
    'the','be','to','of','and','a','in','that','have','it','for','not','on',
    'with','he','as','you','do','at','this','but','his','by','from','they',
    'we','say','her','she','or','an','will','my','one','all','would','there',
    'their','what','so','up','out','if','about','who','get','which','go','me',
    'when','make','can','like','time','no','just','him','know','take','people',
    'into','year','your','good','some','could','them','see','other','than',
    'then','now','look','only','come','its','over','think','also','back','after',
    'use','two','how','our','work','first','well','way','even','new','want',
    'because','any','these','give','day','most','us','between','need','large',
    'often','hand','high','place','hold','turn','where','much','before','move',
    'right','boy','old','too','same','tell','does','set','three','small','put',
    'end','does','another','well','large','big','down','never','start','city',
    'play','small','number','off','always','next','open','seem','together',
    'white','children','begin','got','walk','example','ease','paper','group',
    'always','music','those','both','mark','book','letter','until','mile',
    'river','car','feet','care','second','enough','plain','girl','usual',
    'young','ready','above','ever','red','list','though','feel','talk','bird',
    'soon','body','dog','family','direct','pose','leave','song','measure',
    'door','product','black','short','numeral','class','wind','question',
    'happen','complete','ship','area','half','rock','order','fire','south',
    'problem','piece','told','knew','pass','since','top','whole','king',
    'space','heard','best','hour','better','during','hundred','five','remember',
    'step','early','hold','west','ground','interest','reach','fast','five',
  ],
  code: [
    'function','return','const','let','var','if','else','for','while','do',
    'switch','case','break','continue','class','extends','import','export',
    'default','new','this','null','undefined','true','false','typeof','void',
    'delete','throw','try','catch','finally','async','await','yield','static',
    'public','private','protected','abstract','interface','type','enum','from',
    'string','number','boolean','object','array','promise','callback','event',
    'module','require','console','log','error','warn','info','debug','process',
    'window','document','element','query','selector','style','class','id',
    'props','state','render','component','hook','effect','ref','context',
    'reduce','filter','map','forEach','find','some','every','includes',
    'push','pop','shift','unshift','splice','slice','concat','join','split',
    'length','keys','values','entries','assign','spread','rest','destructure',
    'arrow','template','literal','optional','chaining','nullish','coalescing',
  ],
  quotes: [
    'the quick brown fox jumps over the lazy dog',
    'to be or not to be that is the question',
    'all that glitters is not gold',
    'it always seems impossible until it is done',
    'the only way to do great work is to love what you do',
    'in the middle of difficulty lies opportunity',
    'life is what happens when you are busy making other plans',
    'the future belongs to those who believe in the beauty of their dreams',
    'it does not matter how slowly you go as long as you do not stop',
    'success is not final failure is not fatal it is the courage to continue that counts',
  ],
};

// ─── Config ──────────────────────────────────────────────────────────────────
const CONFIG = {
  wordCount: 25,
  mode: 'common',   // 'common' | 'code' | 'quotes'
  timeLimit: null,  // null = word count mode, number = timed mode (seconds)
};

const SETTINGS_FILE = path.join(os.homedir(), '.ttyper.json');

function loadSettings() {
  try {
    const data = JSON.parse(fs.readFileSync(SETTINGS_FILE, 'utf8'));
    if (data.wordCount !== undefined) CONFIG.wordCount = data.wordCount;
    if (data.mode      !== undefined) CONFIG.mode      = data.mode;
    if (data.timeLimit !== undefined) CONFIG.timeLimit = data.timeLimit;
  } catch (_) {}
}

function saveSettings() {
  try {
    fs.writeFileSync(SETTINGS_FILE, JSON.stringify({
      wordCount: CONFIG.wordCount,
      mode:      CONFIG.mode,
      timeLimit: CONFIG.timeLimit,
    }), 'utf8');
  } catch (_) {}
}

// ─── Game state ───────────────────────────────────────────────────────────────
let state = {};

function buildWordList(mode, count) {
  const list = WORD_LISTS[mode];
  if (mode === 'quotes') {
    const q = list[Math.floor(Math.random() * list.length)];
    return q.split(' ');
  }
  const shuffled = [...list].sort(() => Math.random() - 0.5);
  const words = [];
  while (words.length < count) {
    words.push(...shuffled);
  }
  return words.slice(0, count);
}

function initState() {
  const words = buildWordList(CONFIG.mode, CONFIG.wordCount);
  state = {
    words,
    // typed[i] = string of what was typed for word i
    typed: new Array(words.length).fill(''),
    // done[i] = true once word i was submitted
    done: new Array(words.length).fill(false),
    currentWord: 0,
    currentInput: '',
    startTime: null,
    endTime: null,
    started: false,
    finished: false,
    totalKeystrokes: 0,
    correctKeystrokes: 0,
    mode: CONFIG.mode,
    timeLimit: CONFIG.timeLimit,
    timeLeft: CONFIG.timeLimit,
    timerInterval: null,
    showMenu: false,
    menuCursor: 0,
    menuItems: [],
    quoteQueue: [],
  };
  if (CONFIG.timeLimit) ensureWordBuffer();
}

// ─── Layout ───────────────────────────────────────────────────────────────────
const ROWS = {
  header:   2,
  words:    5,
  input:    9,
  stats:    11,
  footer:   14,
};

function cols() { return process.stdout.columns || 80; }
function center(str, width) {
  const len = stripAnsi(str).length;
  const pad = Math.max(0, Math.floor((width - len) / 2));
  return ' '.repeat(pad) + str;
}
function stripAnsi(str) { return str.replace(/\x1b\[[0-9;]*m/g, ''); }

// ─── Rendering ────────────────────────────────────────────────────────────────
function render() {
  const w = cols();

  if (state.showMenu) {
    renderMenu(w);
    return;
  }

  if (state.finished) {
    renderResults(w);
    return;
  }

  renderGame(w);
}

function renderGame(w) {
  ansi.clear();
  ansi.hide();

  // Title
  ansi.moveTo(ROWS.header, 1);
  ansi.clearLine();
  const modeLabel = state.mode === 'common' ? 'words' : state.mode;
  const timePart = state.timeLimit
    ? ` · ${C.accent}${state.timeLeft}s${C.reset}`
    : ` · ${C.accent}${CONFIG.wordCount} words${C.reset}`;
  const header = `${C.bold}${C.accent}ttyper${C.reset}  ${C.dim}${modeLabel}${timePart}`;
  ansi.write(center(header, w));

  // WPM live display
  ansi.moveTo(ROWS.header + 1, 1);
  ansi.clearLine();
  if (state.started) {
    const wpm = calcWPM();
    const acc = calcAccuracy();
    const live = `${C.dim}wpm: ${C.white}${wpm}${C.reset}  ${C.dim}acc: ${C.white}${acc}%${C.reset}`;
    ansi.write(center(live, w));
  } else {
    ansi.write(center(`${C.dim}start typing to begin…${C.reset}`, w));
  }

  // Word display
  renderWordDisplay(w);

  // Hint
  ansi.moveTo(ROWS.stats, 1);
  ansi.clearLine();
  ansi.write(center(`${C.dim}tab: restart  ctrl+c: quit  ctrl+o: menu${C.reset}`, w));

  ansi.flush();
}

function renderWordDisplay(w) {
  const VISIBLE = 5; // words per "line" conceptually
  const GAP = '  ';

  // Build the full word string with coloring
  // We'll wrap into lines based on terminal width
  const maxWidth = Math.min(w - 4, 70);
  const lines = [];
  let line = '';
  let lineWords = [];
  const margin = Math.floor((w - maxWidth) / 2);

  for (let i = 0; i < state.words.length; i++) {
    const word = state.words[i];
    const gap = lineWords.length > 0 ? GAP : '';
    if ((line + gap + word).length > maxWidth && lineWords.length > 0) {
      lines.push(lineWords);
      lineWords = [];
      line = '';
    }
    lineWords.push(i);
    line += (line.length > 0 ? GAP : '') + word;
  }
  if (lineWords.length > 0) lines.push(lineWords);

  // Find which line the current word is on
  let currentLine = 0;
  for (let li = 0; li < lines.length; li++) {
    if (lines[li].includes(state.currentWord)) {
      currentLine = li;
      break;
    }
  }

  // Show 3 lines: current - 1, current, current + 1
  const displayLines = [currentLine - 1, currentLine, currentLine + 1];

  for (let di = 0; di < 3; di++) {
    const row = ROWS.words + di;
    ansi.moveTo(row, 1);
    ansi.clearLine();

    const li = displayLines[di];
    if (li < 0 || li >= lines.length) continue;

    const wordIndices = lines[li];
    let out = '';

    for (let j = 0; j < wordIndices.length; j++) {
      const i = wordIndices[j];
      const word = state.words[i];
      if (j > 0) out += `${C.dim}${GAP}${C.reset}`;

      if (state.done[i]) {
        // Already completed — color each char correct/wrong
        const typed = state.typed[i];
        for (let ci = 0; ci < Math.max(word.length, typed.length); ci++) {
          if (ci >= word.length) {
            out += `${C.wrong}${typed[ci]}${C.reset}`;
          } else if (ci >= typed.length) {
            out += `${C.wrong}${word[ci]}${C.reset}`;
          } else if (typed[ci] === word[ci]) {
            out += `${C.correct}${word[ci]}${C.reset}`;
          } else {
            out += `${C.wrong}${word[ci]}${C.reset}`;
          }
        }
      } else if (i === state.currentWord) {
        // Active word — show typed chars colored, cursor, then remaining dim
        const typed = state.currentInput;
        for (let ci = 0; ci < word.length; ci++) {
          if (ci < typed.length) {
            const color = typed[ci] === word[ci] ? C.correct : C.wrong;
            out += `${color}${word[ci]}${C.reset}`;
          } else if (ci === typed.length) {
            out += `${C.cursor}${word[ci]}${C.reset}`;
          } else {
            out += `${C.pending}${word[ci]}${C.reset}`;
          }
        }
        // Show extra typed chars beyond word length in red
        for (let ci = word.length; ci < typed.length; ci++) {
          out += `${C.wrong}${typed[ci]}${C.reset}`;
        }
      } else {
        // Pending word
        out += `${C.pending}${word}${C.reset}`;
      }
    }

    ansi.write(' '.repeat(margin) + out);
  }
}

function renderResults(w) {
  ansi.clear();
  ansi.show();

  const wpm = calcWPM();
  const acc = calcAccuracy();
  const elapsed = ((state.endTime - state.startTime) / 1000).toFixed(1);
  const correct = state.words.filter((_, i) => state.done[i] && state.typed[i] === state.words[i]).length;
  const wrong = state.words.filter((_, i) => state.done[i] && state.typed[i] !== state.words[i]).length;

  const lines = [
    '',
    `${C.bold}${C.accent}── results ──${C.reset}`,
    '',
    `${C.dim}wpm     ${C.bold}${C.white}${wpm}${C.reset}`,
    `${C.dim}acc     ${C.bold}${C.white}${acc}%${C.reset}`,
    `${C.dim}time    ${C.bold}${C.white}${elapsed}s${C.reset}`,
    `${C.dim}correct ${C.bold}${C.correct}${correct}${C.reset}`,
    `${C.dim}wrong   ${C.bold}${C.wrong}${wrong}${C.reset}`,
    '',
    `${C.dim}tab: restart  ctrl+o: menu  ctrl+c: quit${C.reset}`,
  ];

  const startRow = Math.floor(process.stdout.rows / 2) - Math.floor(lines.length / 2);
  lines.forEach((l, i) => {
    ansi.moveTo(startRow + i, 1);
    ansi.write(center(l, w));
  });
  ansi.flush();
}

function renderMenu(w) {
  ansi.clear();
  ansi.show();

  const allItems = [
    { type: 'mode',  value: 'common' },
    { type: 'mode',  value: 'code'   },
    { type: 'mode',  value: 'quotes' },
    { type: 'words', value: 10  },
    { type: 'words', value: 25  },
    { type: 'words', value: 50  },
    { type: 'words', value: 100 },
    { type: 'time',  value: null },
    { type: 'time',  value: 15  },
    { type: 'time',  value: 30  },
    { type: 'time',  value: 60  },
  ];
  state.menuItems = allItems;

  function isActive(item) {
    if (item.type === 'mode')  return CONFIG.mode === item.value;
    if (item.type === 'words') return CONFIG.wordCount === item.value && CONFIG.timeLimit === null;
    if (item.type === 'time')  return CONFIG.timeLimit === item.value;
  }
  function label(item) {
    if (item.type === 'time') return item.value === null ? 'off' : item.value + 's';
    return String(item.value);
  }

  let idx = 0;
  function itemLine(item) {
    const i = idx++;
    const cursor   = state.menuCursor === i;
    const active   = isActive(item);
    const prefix   = cursor ? `${C.accent}▸ ` : (active ? `${C.correct}· ` : `  `);
    const text     = cursor ? `${C.white}${label(item)}` : (active ? `${C.correct}${label(item)}` : `${C.pending}${label(item)}`);
    return `  ${prefix}${text}${C.reset}`;
  }

  const lines = [
    `${C.bold}${C.accent}── settings ──${C.reset}`,
    '',
    `${C.dim}mode${C.reset}`,
    itemLine(allItems[0]),
    itemLine(allItems[1]),
    itemLine(allItems[2]),
    '',
    `${C.dim}words${C.reset}  ${C.dim}(word count mode)${C.reset}`,
    itemLine(allItems[3]),
    itemLine(allItems[4]),
    itemLine(allItems[5]),
    itemLine(allItems[6]),
    '',
    `${C.dim}time${C.reset}   ${C.dim}(timed mode)${C.reset}`,
    itemLine(allItems[7]),
    itemLine(allItems[8]),
    itemLine(allItems[9]),
    itemLine(allItems[10]),
    '',
    `${C.dim}↑↓: navigate  enter: select  esc: close  tab: restart${C.reset}`,
  ];

  const startRow = Math.floor(process.stdout.rows / 2) - Math.floor(lines.length / 2);
  lines.forEach((l, i) => {
    ansi.moveTo(startRow + i, 1);
    ansi.write(center(l, w));
  });
  ansi.flush();
}

// ─── Calculations ─────────────────────────────────────────────────────────────
function calcWPM() {
  if (!state.startTime) return 0;
  const elapsed = ((state.endTime || Date.now()) - state.startTime) / 1000 / 60;
  if (elapsed === 0) return 0;
  // Standard: 5 chars = 1 word
  const chars = state.typed.reduce((s, t) => s + t.length, 0);
  return Math.round((chars / 5) / elapsed);
}

function calcAccuracy() {
  if (state.totalKeystrokes === 0) return 100;
  return Math.round((state.correctKeystrokes / state.totalKeystrokes) * 100);
}

// ─── Input handling ───────────────────────────────────────────────────────────
function handleKey(key) {
  // ctrl+c
  if (key === '\x03') {
    cleanup();
    process.exit(0);
  }

  // tab = restart
  if (key === '\x09') {
    if (state.timerInterval) clearInterval(state.timerInterval);
    initState();
    render();
    return;
  }

  // ctrl+o = menu toggle
  if (key === '\x0f') {
    state.showMenu = !state.showMenu;
    render();
    return;
  }

  // esc = close menu if open
  if (key === '\x1b' || key === '\x1b\x1b') {
    if (state.showMenu) {
      state.showMenu = false;
      render();
    }
    return;
  }

  if (state.showMenu) {
    handleMenuKey(key);
    return;
  }

  if (state.finished) return;

  // Start timer on first keystroke
  if (!state.started) {
    state.started = true;
    state.startTime = Date.now();
    if (state.timeLimit) {
      state.timeLeft = state.timeLimit;
      state.timerInterval = setInterval(() => {
        state.timeLeft--;
        if (state.timeLeft <= 0) {
          clearInterval(state.timerInterval);
          finishGame();
        } else {
          render();
        }
      }, 1000);
    }
  }

  const word = state.words[state.currentWord];

  // Backspace
  if (key === '\x7f' || key === '\x08') {
    if (state.currentInput.length > 0) {
      state.currentInput = state.currentInput.slice(0, -1);
    } else if (state.currentWord > 0) {
      state.currentWord--;
      state.currentInput = state.typed[state.currentWord];
      state.done[state.currentWord] = false;
      state.typed[state.currentWord] = '';
    }
    render();
    return;
  }

  // Space = submit current word
  if (key === ' ') {
    if (state.currentInput.length === 0) return; // ignore leading space
    submitWord();
    return;
  }

  // Only printable ASCII
  if (key.length === 1 && key >= ' ') {
    state.totalKeystrokes++;
    const expectedChar = word[state.currentInput.length];
    if (key === expectedChar) {
      state.correctKeystrokes++;
    }
    state.currentInput += key;

    // Auto-advance if input matches word exactly and next char would be space
    // (optional: remove if you prefer explicit space)
    render();
    return;
  }
}

function handleMenuKey(key) {
  const total = state.menuItems.length;

  // Arrow up
  if (key === '\x1b[A') {
    state.menuCursor = (state.menuCursor - 1 + total) % total;
    render();
    return;
  }

  // Arrow down
  if (key === '\x1b[B') {
    state.menuCursor = (state.menuCursor + 1) % total;
    render();
    return;
  }

  // Enter = apply selection
  if (key === '\r' || key === '\n') {
    const item = state.menuItems[state.menuCursor];
    if (item.type === 'mode') {
      CONFIG.mode = item.value;
    } else if (item.type === 'words') {
      CONFIG.wordCount = item.value;
      CONFIG.timeLimit = null;
    } else if (item.type === 'time') {
      CONFIG.timeLimit = item.value;
    }
    saveSettings();
    render();
    return;
  }

  render();
}

function nextQuoteWords() {
  if (state.quoteQueue.length === 0) {
    const indices = WORD_LISTS.quotes.map((_, i) => i);
    state.quoteQueue = indices.sort(() => Math.random() - 0.5);
  }
  const idx = state.quoteQueue.pop();
  return WORD_LISTS.quotes[idx].split(' ');
}

function ensureWordBuffer() {
  const BUFFER = 30;
  const remaining = state.words.length - state.currentWord;
  if (remaining >= BUFFER) return;

  if (CONFIG.mode === 'quotes') {
    while (state.words.length - state.currentWord < BUFFER) {
      const words = nextQuoteWords();
      state.words.push(...words);
      state.typed.push(...new Array(words.length).fill(''));
      state.done.push(...new Array(words.length).fill(false));
    }
  } else {
    const extra = buildWordList(CONFIG.mode, BUFFER - remaining);
    state.words.push(...extra);
    state.typed.push(...new Array(extra.length).fill(''));
    state.done.push(...new Array(extra.length).fill(false));
  }
}

function submitWord() {
  const i = state.currentWord;
  state.typed[i] = state.currentInput;
  state.done[i] = true;
  state.currentInput = '';
  state.currentWord++;

  if (!state.timeLimit && state.currentWord >= state.words.length) {
    finishGame();
    return;
  }

  if (state.timeLimit) ensureWordBuffer();

  render();
}

function finishGame() {
  state.endTime = Date.now();
  state.finished = true;
  if (state.timerInterval) clearInterval(state.timerInterval);
  render();
}

// ─── Cleanup ──────────────────────────────────────────────────────────────────
function cleanup() {
  if (state.timerInterval) clearInterval(state.timerInterval);
  ansi.show();
  process.stdout.write('\n');
  if (process.stdin.setRawMode) process.stdin.setRawMode(false);
  process.stdin.pause();
}

// ─── Entry point ─────────────────────────────────────────────────────────────
function main() {
  loadSettings();

  // Parse CLI args (before TTY check so --help works anywhere)
  const args = process.argv.slice(2);
  for (let i = 0; i < args.length; i++) {
    switch (args[i]) {
      case '-m': case '--mode':
        if (WORD_LISTS[args[i+1]]) CONFIG.mode = args[++i];
        break;
      case '-n': case '--words':
        CONFIG.wordCount = parseInt(args[++i]) || 25;
        break;
      case '-t': case '--time':
        CONFIG.timeLimit = parseInt(args[++i]) || 30;
        break;
      case '-h': case '--help':
        console.log([
          'ttyper — terminal typing speed test',
          '',
          'Usage: node ttyper.js [options]',
          '',
          '  -m, --mode   <mode>   Word list: common, code, quotes  (default: common)',
          '  -n, --words  <n>      Number of words  (default: 25)',
          '  -t, --time   <secs>   Timed mode in seconds (overrides -n)',
          '  -h, --help            Show this help',
          '',
          'Controls:',
          '  space         submit word',
          '  backspace     delete char',
          '  tab           restart',
          '  ctrl+o        open settings menu',
          '  ctrl+c        quit',
        ].join('\n'));
        process.exit(0);
    }
  }

  if (!process.stdin.isTTY) {
    console.error('ttyper requires an interactive terminal (TTY).');
    process.exit(1);
  }

  initState();

  process.stdin.setRawMode(true);
  process.stdin.resume();
  process.stdin.setEncoding('utf8');
  process.stdin.on('data', handleKey);
  process.stdout.on('resize', render);

  process.on('SIGINT', () => { cleanup(); process.exit(0); });
  process.on('exit', cleanup);

  render();
}

main();
