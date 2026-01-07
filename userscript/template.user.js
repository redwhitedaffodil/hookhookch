// ==UserScript==
// @name         Chesshook Generated
// @namespace    http://tampermonkey.net/
// @version      1.0
// @description  Chess.com engine assistance
// @author       ChessHook
// @match        https://www.chess.com/*
// @grant        none
// @require      https://raw.githubusercontent.com/0mlml/chesshook/master/betafish.js
// ==/UserScript==

(() => {
    'use strict';

    const config = {
        engine: '{{.Engine}}',           // 'betafish', 'random', 'external'
        autoMove: {{.AutoMove}},         // true or false
        arrowColor: '{{.ArrowColor}}',   // hex color for arrows
        externalEngineURL: '{{.ExternalEngineURL}}',
        externalEnginePassKey: '{{.PassKey}}'
    };

    let lastFen = '';
    let isThinking = false;
    let worker = null;
    let externalWs = null;

    // Initialize workers based on engine type
    function initEngine() {
        if (config.engine === 'betafish') {
            const betafishWorkerFunc = () => {
                self.importScripts('https://raw.githubusercontent.com/0mlml/chesshook/master/betafish.js');
                self.addEventListener('message', e => {
                    if (e.data.type === 'GETMOVE') {
                        const move = self.getBestMove(e.data.payload.fen, e.data.payload.depth || 10);
                        self.postMessage({ type: 'BESTMOVE', payload: { move: move } });
                    }
                });
            };
            const blob = new Blob(['(' + betafishWorkerFunc.toString() + ')()'], { type: 'application/javascript' });
            worker = new Worker(URL.createObjectURL(blob));
            worker.onmessage = handleWorkerMessage;
        } else if (config.engine === 'external') {
            initExternalEngine();
        }
    }

    function initExternalEngine() {
        const externalWorkerFunc = () => {
            self.ws = null;
            self.authenticated = false;
            self.locked = false;
            self.subscribed = false;
            self.currentFen = '';

            self.connect = (url, passkey) => {
                self.ws = new WebSocket(url);
                self.ws.onopen = () => {
                    console.log('Connected to external engine');
                };
                self.ws.onmessage = (event) => {
                    const msg = event.data;
                    console.log('Engine:', msg);
                    
                    if (msg === 'whoareyou') {
                        self.ws.send('iam chesshook');
                    } else if (msg === 'auth required') {
                        self.ws.send('auth ' + passkey);
                    } else if (msg === 'auth success') {
                        self.authenticated = true;
                    } else if (msg.startsWith('bestmove')) {
                        const move = msg.split(' ')[1];
                        self.postMessage({ type: 'BESTMOVE', payload: { move: move } });
                        self.ws.send('unlock');
                        self.ws.send('unsub');
                        self.locked = false;
                        self.subscribed = false;
                    }
                };
                self.ws.onerror = (error) => {
                    console.error('WebSocket error:', error);
                };
                self.ws.onclose = () => {
                    console.log('Disconnected from external engine');
                    setTimeout(() => self.connect(url, passkey), 5000);
                };
            };

            self.addEventListener('message', e => {
                if (e.data.type === 'CONNECT') {
                    self.connect(e.data.payload.url, e.data.payload.passkey);
                } else if (e.data.type === 'GETMOVE') {
                    if (!self.ws || self.ws.readyState !== WebSocket.OPEN) {
                        console.error('WebSocket not connected');
                        return;
                    }
                    self.currentFen = e.data.payload.fen;
                    self.ws.send('lock');
                    setTimeout(() => {
                        self.ws.send('sub');
                        self.ws.send('position fen ' + self.currentFen);
                        self.ws.send('go movetime ' + (e.data.payload.thinkTime || 2000));
                    }, 100);
                }
            });
        };

        const blob = new Blob(['(' + externalWorkerFunc.toString() + ')()'], { type: 'application/javascript' });
        worker = new Worker(URL.createObjectURL(blob));
        worker.onmessage = handleWorkerMessage;
        worker.postMessage({
            type: 'CONNECT',
            payload: {
                url: config.externalEngineURL,
                passkey: config.externalEnginePassKey
            }
        });
    }

    function handleWorkerMessage(e) {
        if (e.data.type === 'BESTMOVE') {
            const move = e.data.payload.move;
            isThinking = false;
            
            if (move) {
                highlightMove(move);
                if (config.autoMove) {
                    executeMove(move);
                }
            }
        }
    }

    function getEngineMove(fen) {
        if (isThinking) return;
        isThinking = true;

        if (config.engine === 'random') {
            // Random move for testing
            setTimeout(() => {
                isThinking = false;
                console.log('Random move mode');
            }, 500);
        } else if (worker) {
            worker.postMessage({
                type: 'GETMOVE',
                payload: {
                    fen: fen,
                    depth: 10,
                    thinkTime: 2000
                }
            });
        }
    }

    function highlightMove(move) {
        const board = document.querySelector('wc-chess-board') || document.querySelector('chess-board');
        if (!board) return;

        // Parse move (e.g., "e2e4" to from:"e2", to:"e4")
        const from = move.substring(0, 2);
        const to = move.substring(2, 4);

        // Draw arrow
        if (board.setShapes) {
            board.setShapes([{
                brush: config.arrowColor,
                orig: from,
                dest: to
            }]);
        }
    }

    function executeMove(move) {
        const board = document.querySelector('wc-chess-board') || document.querySelector('chess-board');
        if (!board || !board.game) return;

        try {
            const from = move.substring(0, 2);
            const to = move.substring(2, 4);
            board.game.move({ from: from, to: to });
        } catch (e) {
            console.error('Failed to execute move:', e);
        }
    }

    function updateLoop() {
        const board = document.querySelector('wc-chess-board') || document.querySelector('chess-board');
        if (!board || !board.game) return;

        const fen = board.game.getFEN();
        if (fen === lastFen || isThinking) return;

        lastFen = fen;
        getEngineMove(fen);
    }

    // Initialize
    console.log('ChessHook userscript loaded');
    console.log('Engine:', config.engine);
    console.log('Auto-move:', config.autoMove);
    
    initEngine();
    setInterval(updateLoop, 200);
})();
