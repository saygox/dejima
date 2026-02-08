import { writable } from 'svelte/store';

export interface ConnectionState {
  serialPort: string;
  serialConnected: boolean;
  videoStreaming: boolean;
}

export const connection = writable<ConnectionState>({
  serialPort: '',
  serialConnected: false,
  videoStreaming: false,
});

export function updateSerial(port: string) {
  connection.update(s => ({
    ...s,
    serialPort: port,
    serialConnected: port !== '',
  }));
}

export function updateVideo(streaming: boolean) {
  connection.update(s => ({
    ...s,
    videoStreaming: streaming,
  }));
}
