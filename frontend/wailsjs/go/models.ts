export namespace audio {
	
	export class AudioDevice {
	    id: string;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new AudioDevice(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	    }
	}

}

export namespace config {
	
	export class Config {
	    serial_port: string;
	    baud_rate: number;
	    device_index: number;
	    device_path: string;
	    jpeg_quality: number;
	    capture_width: number;
	    capture_height: number;
	    audio_device_id: string;
	    audio_volume: number;
	    audio_muted: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.serial_port = source["serial_port"];
	        this.baud_rate = source["baud_rate"];
	        this.device_index = source["device_index"];
	        this.device_path = source["device_path"];
	        this.jpeg_quality = source["jpeg_quality"];
	        this.capture_width = source["capture_width"];
	        this.capture_height = source["capture_height"];
	        this.audio_device_id = source["audio_device_id"];
	        this.audio_volume = source["audio_volume"];
	        this.audio_muted = source["audio_muted"];
	    }
	}

}

export namespace video {
	
	export class VideoDevice {
	    index: number;
	    name: string;
	    path: string;
	
	    static createFrom(source: any = {}) {
	        return new VideoDevice(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.name = source["name"];
	        this.path = source["path"];
	    }
	}

}

