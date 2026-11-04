importScripts("/wasm_exec.js");

const ready = (async () => {
    const go = new Go();
    const resp = await fetch("/crypto.wasm");

    let result;
    try {
        result = await WebAssembly.instantiateStreaming(resp.clone(), go.importObject);
    } catch {
        result = await WebAssembly.instantiate(await resp.arrayBuffer(), go.importObject);
    }

    go.run(result.instance); // never resolves, so don't await it

    for (let i = 0; i < 500 && !self.vaultCrypto; i++) {
        await new Promise((r) => setTimeout(r, 10));
    }
    if (!self.vaultCrypto) throw new Error("vaultCrypto did not initialise");
})();

self.onmessage = async (e) => {
    const { id, method, args } = e.data;
    try {
        await ready;
        const fn = self.vaultCrypto[method];
        if (typeof fn !== "function") throw new Error("unknown method: " + method);
        const result = await fn(...args);
        self.postMessage({ id, ok: true, result });
    } catch (err) {
        self.postMessage({ id, ok: false, error: String((err && err.message) || err) });
    }
};