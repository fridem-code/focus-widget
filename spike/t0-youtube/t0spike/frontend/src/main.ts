// Spike T0 page: drives the YouTube IFrame Player and reports what happened to the
// Go side over plain HTTP, so the same page works from wails.localhost and 127.0.0.1.

declare global {
    interface Window {
        YT: any;
        onYouTubeIframeAPIReady: () => void;
    }
}

const ARRAY_PLAYLIST = ['M7lc1UVf-VE', 'dQw4w9WgXcQ'];
const INVALID_ID = 'invalid0000';

const params = new URLSearchParams(location.search);
const LIVE_ID = params.get('live') ?? '0sfDth4lMLc';
const variant = params.get('variant') ?? 'unknown';
const port = params.get('port');
const selftest = params.get('selftest') === 'true';
const reportBase = port ? `http://127.0.0.1:${port}` : '';
const logEl = document.getElementById('log')!;
document.getElementById('title')!.textContent = `t0spike · ${variant} · ${location.origin}`;

const errors: { code: number; videoId: string }[] = [];
let player: any;

function report(key: string, value: unknown) {
    logEl.textContent += `${key}: ${JSON.stringify(value)}\n`;
    logEl.scrollTop = logEl.scrollHeight;
    // text/plain keeps this a "simple" request, so no CORS preflight is needed.
    fetch(`${reportBase}/report`, {
        method: 'POST',
        headers: {'Content-Type': 'text/plain'},
        body: JSON.stringify({variant, key, value, t: Math.round(performance.now())}),
    }).catch(() => {});
}

const sleep = (ms: number) => new Promise(r => setTimeout(r, ms));

function snapshot() {
    const data = player.getVideoData?.() ?? {};
    return {
        state: player.getPlayerState(),
        time: Math.round(player.getCurrentTime() * 10) / 10,
        muted: player.isMuted(),
        volume: player.getVolume(),
        videoId: data.video_id,
        isLive: data.isLive,
        playlistIndex: player.getPlaylistIndex?.(),
    };
}

// Playing means state 1 and the playhead actually moves (an ad also counts as audio playing).
async function waitPlaying(timeoutMs: number) {
    const deadline = Date.now() + timeoutMs;
    let last = -1;
    while (Date.now() < deadline) {
        await sleep(1000);
        const t = player.getCurrentTime();
        if (player.getPlayerState() === 1 && last >= 0 && t > last) {
            return true;
        }
        last = t;
    }
    return false;
}

async function runSelftest() {
    report('env', {
        origin: location.origin,
        href: location.href,
        userAgent: navigator.userAgent,
        hasBeenActive: (navigator as any).userActivation?.hasBeenActive,
    });

    player.playVideo();
    const unmuted = await waitPlaying(15000);
    report('liveUnmutedNoGesture', {ok: unmuted, ...snapshot()});
    if (!unmuted) {
        player.mute();
        player.playVideo();
        report('liveMutedNoGesture', {ok: await waitPlaying(15000), ...snapshot()});
        player.unMute();
        await sleep(1500);
        report('unmuteAfterMutedPlay', snapshot());
    }

    player.setVolume(30);
    await sleep(500);
    report('setVolume30', {ok: player.getVolume() === 30, volume: player.getVolume()});

    player.pauseVideo();
    await sleep(1500);
    report('pause', {ok: player.getPlayerState() === 2, ...snapshot()});

    player.loadPlaylist(ARRAY_PLAYLIST);
    const listPlaying = await waitPlaying(15000);
    const before = snapshot();
    player.nextVideo();
    await sleep(4000);
    const after = snapshot();
    report('arrayPlaylistNext', {
        ok: listPlaying && after.playlistIndex === 1 && after.videoId !== before.videoId,
        before,
        after,
    });

    player.setShuffle(true);
    report('setShuffle', {ok: true});

    const errorsBefore = errors.length;
    player.loadVideoById(INVALID_ID);
    await sleep(8000);
    report('invalidVideo', {gotError: errors.length > errorsBefore, errors: errors.slice(errorsBefore)});

    report('allErrors', errors);
    report('done', true);
}

function wireButtons() {
    const on = (id: string, fn: () => void) => document.getElementById(id)!.addEventListener('click', fn);
    on('play', () => player.playVideo());
    on('pause', () => player.pauseVideo());
    on('next', () => player.nextVideo());
    on('live', () => player.loadVideoById(LIVE_ID));
    on('list', () => player.loadPlaylist(ARRAY_PLAYLIST));
    document.getElementById('volume')!.addEventListener('input', e => {
        player.setVolume(Number((e.target as HTMLInputElement).value));
    });
}

window.onYouTubeIframeAPIReady = () => {
    player = new window.YT.Player('player', {
        width: 356,
        height: 200,
        videoId: LIVE_ID,
        playerVars: {playsinline: 1, origin: location.origin, enablejsapi: 1},
        events: {
            onReady: () => {
                report('ready', snapshot());
                wireButtons();
                if (selftest) {
                    runSelftest().catch(err => {
                        report('selftestException', String(err));
                        report('done', true);
                    });
                }
            },
            onStateChange: (e: any) => report('state', e.data),
            onError: (e: any) => {
                errors.push({code: e.data, videoId: player.getVideoData?.().video_id});
                report('error', e.data);
            },
            onAutoplayBlocked: () => report('autoplayBlocked', true),
        },
    });
};

if (port) {
    const echo = document.createElement('iframe');
    echo.hidden = true;
    echo.src = `${reportBase}/echo?variant=${variant}`;
    document.body.appendChild(echo);
}

const api = document.createElement('script');
api.src = 'https://www.youtube.com/iframe_api';
api.onerror = () => {
    report('iframeApiLoadFailed', true);
    report('done', true);
};
document.head.appendChild(api);

export {};
