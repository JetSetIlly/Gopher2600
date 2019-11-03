
if ('function' == typeof importScripts) {
	importScripts('wasm_exec.js');
	importScripts('base64js.min.js');

	const go = new Go();
	WebAssembly.instantiateStreaming(fetch("web2600.wasm"), go.importObject).then((result) => {
		go.run(result.instance);
	});

	function log(msg) {
		self.postMessage({cmd: 'log', msg: msg});
	}

	function updateDebug(target, value) {
		self.postMessage({cmd: 'updateDebug', target: target, value: value});
	}

	function updateCanvas(encodedImage) {
		var a = base64js.toByteArray(encodedImage);
		var decodedImage = new Uint8ClampedArray(a).buffer;
		self.postMessage({cmd: "updateCanvas", image: decodedImage});
	}

	function updateCanvasSize(width, height) {
		self.postMessage({cmd: "updateCanvasSize", width: width, height: height});
	}
}
