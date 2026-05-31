package io.github.smartbird.dispatcher.controller;

import io.github.smartbird.dispatcher.components.lot.LotRegistry;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;
import org.telegram.telegrambots.meta.api.methods.updatingmessages.EditMessageText;
import org.telegram.telegrambots.meta.exceptions.TelegramApiException;

@RestController
@RequestMapping("/api/internal/lots")
@RequiredArgsConstructor
@Slf4j
public class LotNotificationController {

    private final LotRegistry registry;
    private final AuctionBot bot;

    @PostMapping("/{lotId}/finish")
    public ResponseEntity<Void> finishLot(@PathVariable String lotId, @RequestBody LotFinishRequest request) {
        // Пытаемся перевести статус в FINISHED. Если лот не найден или уже закрыт — отказываем
        if (!registry.markFinished(lotId)) {
            return ResponseEntity.status(HttpStatus.NOT_FOUND).build();
        }

        LotRegistry.LotState state = registry.get(lotId);
        try {
            // Убираем inline-клавиатуру, передавая null
            EditMessageText edit = EditMessageText.builder()
                    .chatId(state.chatId())
                    .messageId(state.messageId())
                    .text(String.format("Лот #%s\n✅ Победитель: %s\n💰 Финальная цена: %.2f\n🏁 Аукцион завершён.",
                            lotId, request.winnerId(), request.finalPrice()))
                    .replyMarkup(null)
                    .build();

            bot.execute(edit);
            return ResponseEntity.ok().build();
        } catch (TelegramApiException e) {
            log.error("Не удалось отредактировать сообщение лота {}: {}", lotId, e.getMessage());
            // При ошибке Telegram статус остаётся FINISHED в памяти, чтобы избежать бесконечных ретраев
            return ResponseEntity.status(HttpStatus.INTERNAL_SERVER_ERROR).build();
        }
    }

    public record LotFinishRequest(String winnerId, double finalPrice) {}
}