package io.github.smartbird.dispatcher.client;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.core.io.ByteArrayResource;
import org.springframework.http.HttpEntity;
import org.springframework.http.HttpHeaders;
import org.springframework.http.MediaType;
import org.springframework.http.client.SimpleClientHttpRequestFactory;
import org.springframework.stereotype.Component;
import org.springframework.util.LinkedMultiValueMap;
import org.springframework.util.MultiValueMap;
import org.springframework.web.client.RestTemplate;

import java.util.Map;

@Component
public class BackendClient {
    private final RestTemplate restTemplate;
    private final String backendUrl;

    public BackendClient(@Value("${backend.url}") String backendUrl) {
        this.backendUrl = backendUrl;
        SimpleClientHttpRequestFactory factory = new SimpleClientHttpRequestFactory();
        factory.setConnectTimeout(3000);
        factory.setReadTimeout(5000);
        this.restTemplate = new RestTemplate(factory);
    }

    public LotResponse createLot(Long userId, String description, byte[] photoBytes) {
        HttpHeaders headers = new HttpHeaders();
        headers.setContentType(MediaType.MULTIPART_FORM_DATA);

        MultiValueMap<String, Object> body = new LinkedMultiValueMap<>();
        body.add("userId", String.valueOf(userId));
        body.add("description", description);
        // Обёртка для отправки байтов как файла
        body.add("photo", new ByteArrayResource(photoBytes) {
            @Override public String getFilename() { return "lot.jpg"; }
        });

        HttpEntity<MultiValueMap<String, Object>> request = new HttpEntity<>(body, headers);
        return restTemplate.postForObject(backendUrl + "/api/lots", request, LotResponse.class);
    }

    public record LotResponse(String lotId, double price, String deadline) {}

    public BidResponse placeBid(String lotId, Long userId) {
        HttpHeaders headers = new HttpHeaders();
        headers.setContentType(MediaType.APPLICATION_JSON);
        // Формируем JSON тело запроса
        var body = Map.of("userId", userId);
        HttpEntity<Map<String, Long>> request = new HttpEntity<>(body, headers);

        return restTemplate.postForObject(
                backendUrl + "/api/lots/{lotId}/bid", request, BidResponse.class, lotId);
    }

    public record BidResponse(double price, String status) {}
}
