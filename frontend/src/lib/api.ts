import * as App from '../../wailsjs/go/app/App';
import { app, publishing } from '../../wailsjs/go/models';

export { App as api, app, publishing };

/** ActionError preserves a structured backend ActionResult. */
export class ActionError extends Error {
  readonly result: app.ActionResult;

  constructor(result: app.ActionResult) {
    super(result.message || 'لم تكتمل العملية.');
    this.name = 'ActionError';
    this.result = result;
  }
}

/** runAction rejects when the backend reports a structured failure. */
export async function runAction(promise: Promise<app.ActionResult>): Promise<app.ActionResult> {
  const result = await promise;
  if (!result.ok) {
    throw new ActionError(result);
  }
  return result;
}

/** publicationAction is the typed equivalent for publish/unpublish results. */
export async function publicationAction(
  promise: Promise<app.PublicationResultDTO>,
): Promise<app.PublicationResultDTO> {
  const result = await promise;
  if (!result.ok) {
    throw new ActionError({
      ok: false,
      code: result.code,
      field: result.field,
      message: result.message,
      entityId: 0,
    });
  }
  return result;
}

export function errorMessage(error: unknown): string {
  if (error instanceof ActionError) {
    return error.result.message;
  }
  if (typeof error === 'string') {
    return error || 'لم تكتمل العملية.';
  }
  if (error instanceof Error) {
    return error.message || 'لم تكتمل العملية.';
  }
  return 'لم تكتمل العملية.';
}

export function isActionCode(error: unknown, code: string): boolean {
  return error instanceof ActionError && error.result.code === code;
}
