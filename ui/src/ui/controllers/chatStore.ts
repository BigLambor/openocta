import { ReactiveController, ReactiveControllerHost } from "lit";
import type { ChatAttachment, ChatQueueItem } from "../ui-types.ts";

export class ChatStore implements ReactiveController {
  host: ReactiveControllerHost;

  chatLoading = false;
  chatSending = false;
  chatMessage = "";
  chatMessages: unknown[] = [];
  chatToolMessages: unknown[] = [];
  chatStream: string | null = null;
  chatStreamStartedAt: number | null = null;
  chatRunId: string | null = null;
  chatAvatarUrl: string | null = null;
  chatThinkingLevel: string | null = null;
  chatModelRef: string | null = null;
  chatQueue: ChatQueueItem[] = [];
  chatAttachments: ChatAttachment[] = [];

  constructor(host: ReactiveControllerHost) {
    this.host = host;
    host.addController(this);
  }

  hostConnected() {}
  hostDisconnected() {}
}
