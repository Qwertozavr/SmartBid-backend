package io.github.smartbird.dispatcher.controller;

import io.github.smartbird.dispatcher.client.BackendClient;
import io.github.smartbird.dispatcher.components.lot.LotRegistry;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;
import org.springframework.web.client.RestTemplate;
import org.telegram.telegrambots.bots.DefaultBotOptions;
import org.telegram.telegrambots.bots.TelegramLongPollingBot;
import org.telegram.telegrambots.meta.api.methods.AnswerCallbackQuery;
import org.telegram.telegrambots.meta.api.methods.GetFile;
import org.telegram.telegrambots.meta.api.methods.send.SendMessage;
import org.telegram.telegrambots.meta.api.methods.updatingmessages.EditMessageText;
import org.telegram.telegrambots.meta.api.objects.CallbackQuery;
import org.telegram.telegrambots.meta.api.objects.Message;
import org.telegram.telegrambots.meta.api.objects.Update;
import org.telegram.telegrambots.meta.api.objects.replykeyboard.InlineKeyboardMarkup;
import org.telegram.telegrambots.meta.api.objects.replykeyboard.buttons.InlineKeyboardButton;
import org.telegram.telegrambots.meta.exceptions.TelegramApiException;

import java.util.List;

@Component
@Slf4j
public class AuctionBot extends TelegramLongPollingBot {
    private final BackendClient backendClient;
    private final LotRegistry lotRegistry;
    private final String token;
    private final String username;
    private final Long targetChatId;
    private final RestTemplate downloader;

    public AuctionBot(BackendClient backendClient, LotRegistry lotRegistry,
                      @Value("${telegram.bot.token}") String token,
                      @Value("${telegram.bot.username}") String username,
                      @Value("${telegram.target-chat-id}") Long targetChatId) {
        super(new DefaultBotOptions());
        this.backendClient = backendClient;
        this.lotRegistry = lotRegistry;
        this.token = token;
        this.username = username;
        this.targetChatId = targetChatId;
        this.downloader = new RestTemplate();
    }

    @Override public String getBotToken() { return token; }
    @Override public String getBotUsername() { return username; }

    @Override
    public void onUpdateReceived(Update update) {
        if (update.hasMessage() && update.getMessage().hasText()) {
            String text = update.getMessage().getText();
            if (text.startsWith("/start") || text.startsWith("/help")) {
                handleStartCommand(update.getMessage());
                return;
            }
        }
        if (update.hasMessage() && update.getMessage().hasPhoto()
                && update.getMessage().getCaption() != null && !update.getMessage().getCaption().isEmpty()) {
            handleLotCreation(update.getMessage());
        }
        if (update.hasCallbackQuery()) {
            handleCallbackQuery(update.getCallbackQuery());
        }
//        // Проверка на наличие фото и подписи (caption)
//        if (update.hasMessage() && update.getMessage().hasPhoto()
//                && update.getMessage().getCaption() != null && !update.getMessage().getCaption().isEmpty()) {
//            handleLotCreation(update.getMessage());
//        }
        // CallbackQuery игнорируется до Шага 2
    }

    private void handleStartCommand(Message message) {
        String helpText = """
            🏁 Добро пожаловать в аукционный сервис!

            📦 Чтобы создать лот:
            1. Отправьте фото товара
            2. Добавьте описание в подписи к фото

            📈 После создания:
            • Лот появится в канале с кнопкой «Повысить ставку»
            • Ставка увеличивается на 5% при нажатии
            • Аукцион длится 1 час
            • Победитель будет указан в сообщении лота

            ⚠️ Важно: отправляйте фото и описание одним сообщением.
            """;

        SendMessage response = SendMessage.builder()
                .chatId(message.getChatId())
                .text(helpText)
                .build();

        try {
            execute(response);
        } catch (TelegramApiException e) {
            log.error("Ошибка отправки сообщения /start: {}", e.getMessage());
        }
    }

    private void handleLotCreation(Message message) {
        Long userId = message.getFrom().getId();
        String description = message.getCaption();
        String fileId = message.getPhoto().get(message.getPhoto().size() - 1).getFileId();

        try {
            byte[] photoBytes = downloadTelegramFile(fileId);
            BackendClient.LotResponse response = backendClient.createLot(userId, description, photoBytes);

            // Создаём кнопку через builder
            InlineKeyboardButton button = InlineKeyboardButton.builder()
                    .text(String.format("Повысить ставку (%.2f → %.2f)", response.price(), response.price() * 1.05))
                    .callbackData("bid:" + response.lotId())
                    .build();

// Создаём разметку: keyboardRow принимает List<InlineKeyboardButton>
            InlineKeyboardMarkup markup = InlineKeyboardMarkup.builder()
                    .keyboardRow(List.of(button))  // <-- Исправление: оборачиваем кнопку в список
                    .build();

            SendMessage sendMsg = SendMessage.builder()
                    .chatId(targetChatId)
                    .text(String.format("Лот #%s\nТекущая ставка: %.2f\nДедлайн: %s",
                            response.lotId(), response.price(), response.deadline()))
                    .replyMarkup(markup)
                    .build();

            Message sent = execute(sendMsg);
            lotRegistry.register(response.lotId(), sent.getChatId(), sent.getMessageId());
            log.info("Лот создан: {}, chat: {}, msg: {}", response.lotId(), sent.getChatId(), sent.getMessageId());
        } catch (TelegramApiException e) {
            log.error("Ошибка Telegram API при создании лота: {}", e.getMessage());
        } catch (Exception e) {
            log.error("Ошибка взаимодействия с бэкендом: {}", e.getMessage());
        }
    }

    private byte[] downloadTelegramFile(String fileId) throws TelegramApiException {
        org.telegram.telegrambots.meta.api.objects.File file = execute(new GetFile(fileId));
        String url = String.format("https://api.telegram.org/file/bot%s/%s", token, file.getFilePath());
        return downloader.getForObject(url, byte[].class);
    }

    private void handleCallbackQuery(CallbackQuery callback) {
        String callbackId = callback.getId();
        Long userId = callback.getFrom().getId();
        String data = callback.getData();
        Message originalMsg = callback.getMessage();

        AnswerCallbackQuery answer = AnswerCallbackQuery.builder()
                .callbackQueryId(callbackId)
                .build();

        if (originalMsg == null || data == null || !data.startsWith("bid:")) {
            answer.setText("Действие недоступно.");
            try { execute(answer); } catch (TelegramApiException ignored) {}
            return;
        }

        String lotId = data.substring(4);
        Long chatId = originalMsg.getChatId();
        Integer messageId = originalMsg.getMessageId();

        try {
            BackendClient.BidResponse bidResp = backendClient.placeBid(lotId, userId);
            if (bidResp == null) {
                answer.setText("Сервис временно недоступен.");
            } else {
                double newPrice = bidResp.price();
                String status = bidResp.status();

                InlineKeyboardButton button = InlineKeyboardButton.builder()
                        .text(String.format("Повысить ставку (%.2f → %.2f)", newPrice, newPrice * 1.05))
                        .callbackData("bid:" + lotId)
                        .build();

                InlineKeyboardMarkup markup = InlineKeyboardMarkup.builder()
                        .keyboardRow(List.of(button))
                        .build();

                EditMessageText edit = EditMessageText.builder()
                        .chatId(chatId)
                        .messageId(messageId)
                        .text(String.format("Лот #%s\nТекущая ставка: %.2f\nСтатус: %s", lotId, newPrice, status))
                        .replyMarkup(markup)
                        .build();

                try {
                    execute(edit);
                } catch (TelegramApiException e) {
                    boolean isNotModified = e.getMessage().contains("message is not modified") ||
                            e.getMessage().contains("not modified");
                    if (!isNotModified) {
                        log.error("Ошибка редактирования лота {}: {}", lotId, e.getMessage());
                        answer.setText("Не удалось обновить информацию.");
                    }
                }
            }
        } catch (Exception e) {
            log.error("Исключение при обработке ставки для лота {}: {}", lotId, e.getMessage());
            answer.setText("Ошибка обработки ставки.");
        } finally {
            try { execute(answer); } catch (TelegramApiException ignored) {}
        }
    }

}
