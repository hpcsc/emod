import * as vscode from 'vscode';
import { LanguageClient, LanguageClientOptions, ServerOptions } from 'vscode-languageclient/node';

let client: LanguageClient | undefined;

export function activate(_context: vscode.ExtensionContext) {
    const serverOptions: ServerOptions = {
        command: 'emod',
        args: ['lsp'],
    };

    const clientOptions: LanguageClientOptions = {
        documentSelector: [{ language: 'emod' }],
    };

    client = new LanguageClient(
        'emod-lsp',
        'Emod Language Server',
        serverOptions,
        clientOptions,
    );

    client.start().catch((err: unknown) => {
        client = undefined;
        const message =
            err instanceof Error ? err.message : String(err);
        void vscode.window.showErrorMessage(
            `Failed to start Emod language server: ${message}. Ensure emod is installed (see https://github.com/hpcsc/emod) and available on your PATH.`,
        );
    });
}

export function deactivate(): Thenable<void> | undefined {
    if (!client) {
        return undefined;
    }
    return client.stop();
}
