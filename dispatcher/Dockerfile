# Stage 1: Build with Maven
FROM maven:3.9-eclipse-temurin-17 AS build
WORKDIR /app

# Копируем pom.xml первым для кэширования слоя зависимостей
COPY pom.xml .
RUN mvn dependency:go-offline -q

# Копируем исходный код и собираем артефакт
COPY src ./src
RUN mvn clean package -DskipTests -q

# Stage 2: Minimal runtime image
FROM eclipse-temurin:17-jre-alpine AS runtime
WORKDIR /app

# Копируем собранный jar (имя соответствует artifactId + version из pom.xml)
COPY --from=build /app/target/dispatcher-0.0.1-SNAPSHOT.jar app.jar

# Порт, указанный в application.yaml (server.port: 8000)
EXPOSE 8000

# Запуск Spring Boot fat-jar
ENTRYPOINT ["java", "-jar", "app.jar"]