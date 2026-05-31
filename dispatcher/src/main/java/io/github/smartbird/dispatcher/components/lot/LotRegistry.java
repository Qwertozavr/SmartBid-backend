package io.github.smartbird.dispatcher.components.lot;

import org.springframework.stereotype.Component;

import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;

@Component
public class LotRegistry {
    private final Map<String, LotState> registry = new ConcurrentHashMap<>();

    public record LotState(Long chatId, Integer messageId, String status) {}

    public void register(String lotId, Long chatId, Integer messageId) {
        registry.put(lotId, new LotState(chatId, messageId, "ACTIVE"));
    }

    public LotState get(String lotId) { return registry.get(lotId); }
    public boolean exists(String lotId) { return registry.containsKey(lotId); }

    // Атомарно обновляет статус только если лот существует и активен
    public boolean markFinished(String lotId) {
        return registry.computeIfPresent(lotId, (key, current) ->
                "ACTIVE".equals(current.status()) ? new LotState(current.chatId(), current.messageId(), "FINISHED") : current
        ) != null;
    }
}
