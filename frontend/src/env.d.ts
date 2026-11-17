/// <reference types="vite/client" />

interface ImportMetaEnv {
	/** Habilita a página de referência /dev/ui (CP12). */
	readonly OPENJOB_DEV_UI: string;
	readonly VITE_OPENJOB_DEV_UI: string;
}

interface ImportMeta {
	readonly env: ImportMetaEnv;
}
