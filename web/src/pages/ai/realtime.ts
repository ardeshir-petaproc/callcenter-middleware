// Browser-side test of an AI profile against the OpenAI Realtime API over WebRTC,
// authenticated with a short-lived client secret minted by the backend.
import type { ClientSecret } from '../../api/types';

export interface RealtimeLine { speaker: 'caller' | 'ai'; text: string; at: string }
export interface RealtimeHandlers {
  onLine: (l: RealtimeLine) => void;
  onState: (s: 'connecting' | 'live' | 'ended' | 'error', detail?: string) => void;
  onSpeaking?: (speaking: boolean) => void;
}

export interface RealtimeSession { hangup: () => void }

const REALTIME_URL = 'https://api.openai.com/v1/realtime/calls';

export async function startRealtime(secret: ClientSecret, h: RealtimeHandlers): Promise<RealtimeSession> {
  h.onState('connecting');
  const pc = new RTCPeerConnection();
  const audio = document.createElement('audio');
  audio.autoplay = true;
  let mic: MediaStream | null = null;
  let closed = false;

  const hangup = () => {
    if (closed) return;
    closed = true;
    mic?.getTracks().forEach((t) => t.stop());
    try { pc.close(); } catch { /* ignore */ }
    audio.srcObject = null;
    h.onState('ended');
  };

  try {
    pc.ontrack = (e) => { audio.srcObject = e.streams[0]; void audio.play().catch(() => undefined); };
    mic = await navigator.mediaDevices.getUserMedia({ audio: true });
    mic.getTracks().forEach((t) => pc.addTrack(t, mic!));

    const dc = pc.createDataChannel('oai-events');
    dc.onopen = () => h.onState('live');
    dc.onmessage = (ev) => {
      let msg: { type?: string; transcript?: string };
      try { msg = JSON.parse(ev.data as string); } catch { return; }
      const at = new Date().toISOString();
      switch (msg.type) {
        case 'conversation.item.input_audio_transcription.completed':
          if (msg.transcript?.trim()) h.onLine({ speaker: 'caller', text: msg.transcript.trim(), at });
          break;
        case 'response.output_audio_transcript.done':
        case 'response.audio_transcript.done':
          if (msg.transcript?.trim()) h.onLine({ speaker: 'ai', text: msg.transcript.trim(), at });
          break;
        case 'output_audio_buffer.started': h.onSpeaking?.(true); break;
        case 'output_audio_buffer.stopped': h.onSpeaking?.(false); break;
        case 'error': h.onState('error', JSON.stringify((msg as { error?: unknown }).error ?? msg)); break;
        default: break;
      }
    };
    pc.onconnectionstatechange = () => {
      if (pc.connectionState === 'failed' || pc.connectionState === 'disconnected') { h.onState('error', `Connection ${pc.connectionState}`); hangup(); }
    };

    const offer = await pc.createOffer();
    await pc.setLocalDescription(offer);
    const res = await fetch(REALTIME_URL, {
      method: 'POST',
      body: offer.sdp,
      headers: { Authorization: `Bearer ${secret.value}`, 'Content-Type': 'application/sdp' },
    });
    if (!res.ok) throw new Error(`OpenAI rejected the session (${res.status}): ${(await res.text()).slice(0, 200)}`);
    await pc.setRemoteDescription({ type: 'answer', sdp: await res.text() });
  } catch (e) {
    hangup();
    h.onState('error', e instanceof Error ? e.message : String(e));
    throw e;
  }
  return { hangup };
}
