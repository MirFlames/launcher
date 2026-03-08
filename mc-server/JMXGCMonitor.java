import javax.management.*;
import javax.management.remote.*;
import javax.management.openmbean.*;
import java.util.*;

/**
 * Подключается к JMX Minecraft-сервера и выводит статистику GC каждую секунду.
 * Запуск: javac JMXGCMonitor.java && java -cp . JMXGCMonitor
 */
public class JMXGCMonitor {
    public static void main(String[] args) throws Exception {
        String host = args.length > 0 ? args[0] : "127.0.0.1";
        int port = args.length > 1 ? Integer.parseInt(args[1]) : 7091;
        String url = "service:jmx:rmi:///jndi/rmi://" + host + ":" + port + "/jmxrmi";

        System.err.println("Подключение к " + url + " ...");
        JMXServiceURL jmxUrl = new JMXServiceURL(url);
        JMXConnector conn = JMXConnectorFactory.connect(jmxUrl);
        MBeanServerConnection mbsc = conn.getMBeanServerConnection();

        ObjectName gcName = new ObjectName("java.lang:type=GarbageCollector,*");
        Set<ObjectName> gcBeans = mbsc.queryNames(null, gcName);

        System.err.println("Найдено GC: " + gcBeans.size());
        System.err.println("Время | GC | Кол-во сборок | Время в GC (мс) | Heap used (MB)");
        System.err.println("------|----|---------------|-----------------|---------------");

        long start = System.currentTimeMillis();
        while (true) {
            try {
                long totalGcCount = 0;
                long totalGcTime = 0;
                for (ObjectName name : gcBeans) {
                    totalGcCount += (Long) mbsc.getAttribute(name, "CollectionCount");
                    totalGcTime += (Long) mbsc.getAttribute(name, "CollectionTime");
                }

                ObjectName memoryName = new ObjectName("java.lang:type=Memory");
                CompositeData heap = (CompositeData) mbsc.getAttribute(memoryName, "HeapMemoryUsage");
                long used = (Long) heap.get("used");

                long elapsed = (System.currentTimeMillis() - start) / 1000;
                System.out.printf("%5d | %2d | %13d | %15d | %10d%n",
                    elapsed, gcBeans.size(), totalGcCount, totalGcTime, used / 1024 / 1024);
            } catch (Exception e) {
                System.err.println("Ошибка: " + e.getMessage());
            }
            Thread.sleep(1000);
        }
    }
}
