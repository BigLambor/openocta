import { ReactiveController, ReactiveControllerHost } from "lit";
import type { ChannelsStatusSnapshot } from "../types.ts";
import type { NostrProfileFormState } from "../views/channels.nostr-profile-form.ts";

export class ChannelsStore implements ReactiveController {
  host: ReactiveControllerHost;

  channelsLoading = false;
  channelsSnapshot: ChannelsStatusSnapshot | null = null;
  channelsError: string | null = null;
  channelsLastSuccess: number | null = null;
  whatsappLoginMessage: string | null = null;
  whatsappLoginQrDataUrl: string | null = null;
  whatsappLoginConnected: boolean | null = null;
  whatsappBusy = false;
  weworkQrModalOpen = false;
  weworkQrModalLoading = false;
  weworkQrModalPolling = false;
  weworkQrModalSuccess = false;
  weworkQrModalError: string | null = null;
  weworkQrModalReplaceWarn = false;
  weworkQrModalAuthUrl: string | null = null;
  weworkQrModalGenPageUrl: string | null = null;
  weixinQrModalOpen = false;
  weixinQrModalLoading = false;
  weixinQrModalPolling = false;
  weixinQrModalSuccess = false;
  weixinQrModalError: string | null = null;
  weixinQrModalReplaceWarn = false;
  weixinQrModalImageSrc: string | null = null;
  weixinQrModalScanPageUrl: string | null = null;
  weixinQrModalScanned = false;
  nostrProfileFormState: NostrProfileFormState | null = null;
  nostrProfileAccountId: string | null = null;
  channelsSelectedChannelId: string | null = null;

  constructor(host: ReactiveControllerHost) {
    this.host = host;
    host.addController(this);
  }

  hostConnected() {}
  hostDisconnected() {}
}
